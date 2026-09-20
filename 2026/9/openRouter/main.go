// main.go marker - preserve

package main

import (
   "cmp"
   "fmt"
   "io"
   "net/http"
   "os"
   "slices"
   "time"
)

// topN is how many provider rows the output carries: every row is ordered
// by GPQA Diamond descending, the first topN are kept, and those are
// re-sorted by throughput before printing. Hardcoded for now, like
// minContext; it is printed in the output header so the cut is never
// implicit.
const topN = 20

func httpGet(c *http.Client, url string) ([]byte, error) {
   req, err := http.NewRequest(http.MethodGet, url, nil)
   if err != nil {
      return nil, err
   }
   req.Header.Set("User-Agent", "tp-rank/1.0")
   resp, err := c.Do(req)
   if err != nil {
      return nil, err
   }
   defer resp.Body.Close()
   if resp.StatusCode != http.StatusOK {
      return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
   }
   return io.ReadAll(resp.Body)
}

func main() {
   if err := run(); err != nil {
      fmt.Fprintf(os.Stderr, "%v\n", err)
      os.Exit(1)
   }
}

// percentile returns the p-th percentile of ascending-sorted data with
// linear interpolation (numpy's default method): fractional index
// i = (p/100) * (n-1), interpolate between neighbors.
func percentile(sorted []float64, p float64) float64 {
   n := len(sorted)
   if n == 0 {
      return 0
   }
   if n == 1 {
      return sorted[0]
   }
   i := p / 100 * float64(n-1)
   lo := int(i)
   if lo+1 >= n {
      return sorted[n-1]
   }
   frac := i - float64(lo)
   return sorted[lo] + frac*(sorted[lo+1]-sorted[lo])
}

func run() error {
   client := &http.Client{Timeout: 30 * time.Second}

   // --- 1. Catalog: one request, three filters — one server-side (the
   // context floor), two client-side (open weights, dedupe). filterReport
   // names each filter and its effect, so no count is mistaken for the
   // size of the whole catalog.
   cands, stats, err := fetchCandidates(client)
   if err != nil {
      return fmt.Errorf("catalog: %w", err)
   }
   fmt.Fprint(os.Stderr, filterReport(stats))
   if len(cands) == 0 {
      return fmt.Errorf("no candidates after filtering: %d models returned for context >= %d, %d dropped as closed weights, %d as duplicate slugs",
         stats.AfterContext, minContext, stats.DroppedClosed, stats.DroppedDuplicate)
   }

   // --- 2. One block of lines per candidate: the header and the model's
   // context window are printed before the requests so a slow candidate is
   // visible while it is in flight, the outcome lines after the response.
   //
   // A candidate with no data recorded is logged and skipped; any other
   // failure aborts the run — no retries.
   var rows []*row
   skipped := 0
   for i, cd := range cands {
      fmt.Fprintf(os.Stderr, "\n[%d/%d] %s\n", i+1, len(cands), cd.slug)
      fmt.Fprintf(os.Stderr, "  %-30s %d tokens\n", "model context window:", cd.contextLength)

      rs, scoredProviders, err := fetchRows(client, &cd)
      if err != nil {
         if skippable(err) {
            fmt.Fprintf(os.Stderr, "  %-30s %v\n", "skipped:", err)
            skipped++
            continue
         }
         return fmt.Errorf("stats %s: %w", cd.slug, err)
      }

      // Three counts in the order a provider is eliminated: scored on GPQA,
      // of those ranked (one row each, having cleared the throughput join),
      // and the remainder whose endpoint carries no throughput measurement.
      // Printing the second and third together is what makes a zero
      // self-explanatory rather than a bare number.
      fmt.Fprintf(os.Stderr, "  %-30s %d\n", "providers with a GPQA score:", scoredProviders)
      fmt.Fprintf(os.Stderr, "  %-30s %d\n", "provider rows with throughput:", len(rs))
      fmt.Fprintf(os.Stderr, "  %-30s %d\n", "providers without throughput:", scoredProviders-len(rs))
      rows = append(rows, rs...)
   }

   // --- 3. Rank by score, then cut: order every provider row by GPQA
   // Diamond descending and keep the first topN.
   totalRows := len(rows)
   slices.SortFunc(rows, func(a, b *row) int {
      return cmp.Compare(b.GPQA, a.GPQA)
   })
   if len(rows) > topN {
      rows = rows[:topN]
   }

   // A run that ranks nothing is a misconfiguration, not a result: say so
   // rather than printing a bare header. (Delete this block if an empty
   // list should exit 0.)
   if len(rows) == 0 {
      return fmt.Errorf("no provider rows to rank: %d candidates, %d skipped for absent data, %d rows",
         len(cands), skipped, totalRows)
   }

   // The cut floor: the lowest GPQA Diamond score that made the top topN.
   // rows is still score-ordered here, so the last row carries it.
   fmt.Fprintf(os.Stderr, "\nlowest gpqa diamond in top %d: %.1f%%\n", topN, rows[len(rows)-1].GPQA*100)

   // --- 4. Order the kept rows by throughput, descending: the selection
   // above was by score; the printed order is by speed.
   slices.SortFunc(rows, func(a, b *row) int {
      return cmp.Compare(b.Throughput, a.Throughput)
   })

   fmt.Printf("top %d by gpqa diamond score, sorted by: throughput, descending\n\n", topN)

   for _, r := range rows {
      fmt.Printf("model: %s\n", r.Model)
      fmt.Printf("provider: %s\n", r.Provider)
      fmt.Printf("gpqa diamond: %.1f%%\n", r.GPQA*100)
      fmt.Printf("throughput: %.1f tok/s\n", r.Throughput)
      fmt.Println()
   }

   return nil
}

type candidate struct {
   slug string
   // contextLength is the model card's window, echoed in the per-candidate
   // block so the floor's effect is visible per candidate.
   contextLength int
}

// ---------------------------------------------------------------------------
// Rows: one permaslug -> one row per provider
// ---------------------------------------------------------------------------

// providerGPQA aggregates one provider's GPQA entries: every
// gpqa_diamond score plus every endpoint id the provider was scored on.
// Both row dimensions — the GPQA score and the throughput — are reduced by
// the same rule: the median over the provider's scored endpoints.
type providerGPQA struct {
   Scores      []float64
   EndpointIDs []string
}

// ---------------------------------------------------------------------------
// Working types
// ---------------------------------------------------------------------------

// row is one model + one provider: the flat unit of output.
type row struct {
   Model    string
   Provider string
   // GPQA is the median of the provider's AutoExacto GPQA Diamond
   // scores across its scored endpoints, 0..1.
   GPQA float64
   // Throughput is the median of the provider's endpoints' throughput,
   // tokens/sec.
   Throughput float64
}

// fetchRows returns one row per provider: provider, GPQA Diamond score,
// and median throughput, from two per-permaslug stats requests — no model
// page, no HTML. A row is emitted only when the provider has both a GPQA
// score and a throughput measurement. Both the score and the throughput
// are the median over the provider's scored endpoints — one reduction rule
// for both dimensions.
//
// scoredProviders is the number of providers that have a gpqa_diamond score
// for this model, whether or not each ends up with a row. The caller logs it
// beside len(rows), so a candidate that is scored on GPQA but produces
// nothing says why instead of printing a bare zero. It is 0 whenever err is
// non-nil.
//
// A candidate whose stats are simply absent comes back with a skippable
// sentinel; every other failure (transport, HTTP status, decoding) is an
// ordinary error. The throughput request is not made when the scores are
// already absent.
func fetchRows(c *http.Client, cd *candidate) (rows []*row, scoredProviders int, err error) {
   scores, err := fetchScores(c, cd.slug)
   if err != nil {
      return nil, 0, err
   }
   // Endpoint id -> tokens/sec, same uuid namespace as the scores'
   // endpoint_id, so the join below is exact.
   throughputByEndpoint, err := fetchThroughput(c, cd.slug)
   if err != nil {
      return nil, 0, err
   }
   return rowsFromScores(cd, scores, throughputByEndpoint)
}

// rowsFromScores reduces one permaslug's benchmark scores and its
// per-endpoint throughput to rows, one per provider, and reports how many
// providers carried a gpqa_diamond score at all. Scores that carry no
// gpqa_diamond entries (tau data only, say) come back with a skippable
// sentinel.
//
// A provider with a GPQA score whose endpoint has no throughput entry is
// dropped from rows but still counted in scoredProviders. Returning both is
// what lets the caller print "0 ranked of 16 scored" rather than "0 rows".
func rowsFromScores(cd *candidate, scores []scoreEntry, throughputByEndpoint map[string]float64) (rows []*row, scoredProviders int, err error) {
   // GPQA Diamond per provider. A provider can have several scored
   // endpoints (e.g. different quantizations); every score is kept, and
   // all of the provider's endpoint ids are collected for the tps join.
   // The auto-routing pseudo-provider is included: it has a score but a null
   // endpoint id, so it is always counted and never ranked.
   gpqaByProvider := make(map[string]providerGPQA)
   for _, s := range scores {
      if s.BenchmarkType != "gpqa_diamond" {
         continue
      }
      g := gpqaByProvider[s.ProviderName]
      g.Scores = append(g.Scores, s.Score)
      if s.EndpointID != "" && !slices.Contains(g.EndpointIDs, s.EndpointID) {
         g.EndpointIDs = append(g.EndpointIDs, s.EndpointID)
      }
      gpqaByProvider[s.ProviderName] = g
   }
   if len(gpqaByProvider) == 0 {
      return nil, 0, errNoGPQA
   }

   rows = make([]*row, 0, len(gpqaByProvider))
   for provider, g := range gpqaByProvider {
      // Same reduction for both dimensions: the median over the
      // provider's scored endpoints.
      var throughputs []float64
      for _, id := range g.EndpointIDs {
         if tp, ok := throughputByEndpoint[id]; ok {
            throughputs = append(throughputs, tp)
         }
      }
      if len(throughputs) == 0 {
         continue // no throughput for this provider -> no row
      }
      slices.Sort(g.Scores)
      slices.Sort(throughputs)
      rows = append(rows, &row{
         Model:      cd.slug,
         Provider:   provider,
         GPQA:       percentile(g.Scores, 50),
         Throughput: percentile(throughputs, 50),
      })
   }
   return rows, len(gpqaByProvider), nil
}

// main.go marker - preserve
