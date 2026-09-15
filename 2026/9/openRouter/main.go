// main.go marker - preserve

package main

import (
   "cmp"
   "flag"
   "fmt"
   "io"
   "net/http"
   "os"
   "slices"
   "time"
)

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
   minGPQA := flag.Float64("g", 0,
      "drop providers below this GPQA Diamond score, percent (0-100; 0 = no filter)")
   yes := flag.Bool("y", false,
      "confirm: run the fetch (required — without it usage is printed)")
   flag.Parse()

   // -y not used -> flag.Usage as is, and return.
   if !*yes {
      flag.Usage()
      return
   }

   if err := run(*minGPQA); err != nil {
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

func run(minGPQA float64) error {
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

   // --- 2. Providers: sequential, two JSON stats requests per candidate
   // (scores + throughput). A candidate with no data recorded is logged
   // and skipped; any other failure aborts the run — no retries.
   var rows []*row
   skipped := 0
   for i, cd := range cands {
      fmt.Fprintf(os.Stderr, "[%d/%d] %s ctx: %d ",
         i+1, len(cands), cd.slug, cd.contextLength)
      rs, err := fetchRows(client, &cd)
      if err != nil {
         if skippable(err) {
            fmt.Fprintf(os.Stderr, "skip: %v\n", err)
            skipped++
            continue
         }
         fmt.Fprintln(os.Stderr) // terminate the in-progress progress line
         return fmt.Errorf("stats %s: %w", cd.slug, err)
      }
      fmt.Fprintf(os.Stderr, "aa: %.1f\n", cd.intelligence)
      rows = append(rows, rs...)
   }
   fmt.Fprintf(os.Stderr, "skip filter (no data recorded, per candidate):  %6d of %d candidates\n",
      skipped, len(cands))
   fmt.Fprintf(os.Stderr, "rows:                                          %6d\n", len(rows))

   // --- 3. Apply the -g floor: drop providers below minGPQA percent.
   if minGPQA > 0 {
      before := len(rows)
      rows = slices.DeleteFunc(rows, func(r *row) bool {
         return r.GPQA*100 < minGPQA
      })
      fmt.Fprintf(os.Stderr, "  dropped by -g (GPQA below %.1f%%):            %6d\n",
         minGPQA, before-len(rows))
   }

   // A run that ranks nothing is a misconfiguration, not a result: say so
   // rather than printing a bare header. (Delete this block if an empty
   // list should exit 0.)
   if len(rows) == 0 {
      return fmt.Errorf("no rows to rank: %d candidates, %d skipped for absent data, %d rows left after -g",
         len(cands), skipped, len(rows))
   }

   // --- 4. Sort by throughput, descending.
   slices.SortFunc(rows, func(a, b *row) int {
      return cmp.Compare(b.Throughput, a.Throughput)
   })

   fmt.Printf("sorted by: throughput, descending\n\n")

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
   slug         string
   intelligence float64
   // contextLength is the model card's window, echoed in the progress line
   // so the floor's effect is visible per candidate.
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
// A candidate whose stats are simply absent comes back with a skippable
// sentinel; every other failure (transport, HTTP status, decoding) is an
// ordinary error. The throughput request is not made when the scores are
// already absent.
func fetchRows(c *http.Client, cd *candidate) ([]*row, error) {
   scores, err := fetchScores(c, cd.slug)
   if err != nil {
      return nil, err
   }
   // Endpoint id -> tokens/sec, same uuid namespace as the scores'
   // endpoint_id, so the join below is exact.
   throughputByEndpoint, err := fetchThroughput(c, cd.slug)
   if err != nil {
      return nil, err
   }
   return rowsFromScores(cd, scores, throughputByEndpoint)
}

// rowsFromScores reduces one permaslug's benchmark scores and its
// per-endpoint throughput to rows, one per provider. Scores that carry no
// gpqa_diamond entries at all (tau data only, say) come back with a
// skippable sentinel.
func rowsFromScores(cd *candidate, scores []scoreEntry, throughputByEndpoint map[string]float64) ([]*row, error) {
   // GPQA Diamond per provider. A provider can have several scored
   // endpoints (e.g. different quantizations); every score is kept, and
   // all of the provider's endpoint ids are collected for the tps join.
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
      return nil, errNoGPQA
   }

   var rows []*row
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
   return rows, nil
}

// main.go marker - preserve
