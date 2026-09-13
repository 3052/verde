// main.go marker - preserve

package main

import (
   "cmp"
   "errors"
   "flag"
   "fmt"
   "io"
   "net/http"
   "os"
   "slices"
   "strings"
   "time"
)

// errNoGPQA is the one failure fetchRows retries: the page payload
// sometimes arrives without any gpqa_diamond scores.
var errNoGPQA = errors.New("no gpqa_diamond scores in page payload")

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
   minIntelligence := flag.Float64("i", 0,
      "drop candidates below this AA intelligence index (0 = no filter)")
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

   if err := run(*minIntelligence, *minGPQA); err != nil {
      fmt.Fprintf(os.Stderr, "%v\n", err)
      os.Exit(1)
   }
}

// modelPageSlug derives the model page path from a permaslug by
// stripping the trailing version date: "z-ai/glm-5.3-20260816" ->
// "z-ai/glm-5.3". A permaslug without a trailing 8-digit date is
// returned unchanged.
func modelPageSlug(permaslug string) string {
   before, date, found := strings.CutLast(permaslug, "-")
   if !found || len(date) != 8 {
      return permaslug
   }
   for _, r := range date {
      if r < '0' || r > '9' {
         return permaslug
      }
   }
   return before
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

func run(minIntelligence, minGPQA float64) error {
   client := &http.Client{Timeout: 30 * time.Second}

   // --- 1. Catalog: one request, filter to open weights.
   cands, total, err := fetchCandidates(client, minIntelligence)
   if err != nil {
      return fmt.Errorf("catalog: %w", err)
   }
   fmt.Fprintf(os.Stderr, "%d models in catalog, %d open-weights candidates\n",
      total, len(cands))
   if len(cands) == 0 {
      return fmt.Errorf("no candidates matched")
   }

   // --- 2. Providers: sequential, one request per candidate. A failed
   // request aborts the run — no retries. The only exception is a page
   // payload missing its gpqa_diamond scores, which fetchRows retries once.
   var rows []*row
   for i, cd := range cands {
      fmt.Fprintf(os.Stderr, "[%d/%d] %s ", i+1, len(cands), cd.slug)
      rs, err := fetchRows(client, &cd)
      if err != nil {
         fmt.Fprintln(os.Stderr) // terminate the in-progress progress line
         return fmt.Errorf("page %s: %w", cd.slug, err)
      }
      fmt.Fprintf(os.Stderr, "aa: %.1f\n", cd.intelligence)
      rows = append(rows, rs...)
   }

   // --- 3. Apply the -g floor: drop providers below minGPQA percent.
   rows = slices.DeleteFunc(rows, func(r *row) bool {
      return r.GPQA*100 < minGPQA
   })

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
   // pageSlug is the model page path for this candidate (permaslug
   // minus the version date).
   pageSlug string
}

// ---------------------------------------------------------------------------
// Rows: one model page -> one row per provider
// ---------------------------------------------------------------------------

// providerGPQA aggregates one provider's GPQA entries: every
// gpqa_diamond score plus every endpoint id the provider was scored on.
// Both row dimensions — the GPQA score and the p50 throughput — are
// reduced by the same rule: the median over the provider's scored
// endpoints.
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
   // Throughput is the median of the provider's endpoints' p50
   // throughput, tokens/sec.
   Throughput float64
}

// fetchRows returns one row per provider: provider, GPQA Diamond score,
// and median p50 throughput, all from a single model page request. A row
// is emitted only when the provider has both a GPQA score and a tps
// measurement. Both the score and the throughput are the median over the
// provider's scored endpoints — one reduction rule for both dimensions.
//
// The AutoExacto scores are not always present in the response, so a page
// that comes back without any gpqa_diamond scores is fetched one extra
// time; if the second page is missing them too, errNoGPQA is returned.
// Every other failure (transport, HTTP status, decoding) aborts
// immediately — no retries.
func fetchRows(c *http.Client, cd *candidate) ([]*row, error) {
   const attempts = 2
   var lastErr error
   for i := 0; i < attempts; i++ {
      st, err := fetchPageState(c, cd.pageSlug)
      if err != nil {
         return nil, err
      }
      rows, err := rowsFromPageState(cd, st)
      if err == nil {
         return rows, nil
      }
      if !errors.Is(err, errNoGPQA) {
         return nil, err
      }
      lastErr = err
      if i+1 < attempts {
         fmt.Fprintf(os.Stderr, "[%v, retrying] ", err)
      }
   }
   return nil, fmt.Errorf("%w (after %d attempts)", lastErr, attempts)
}

// rowsFromPageState reduces one model page's query state to rows, one per
// provider. It returns errNoGPQA when the page carries no gpqa_diamond
// scores at all.
func rowsFromPageState(cd *candidate, st *pageState) ([]*row, error) {
   // p50 throughput per endpoint id. The two stats queries carry the
   // same array, so identical entries overwrite each other.
   throughputByEndpoint := make(map[string]float64)
   for _, e := range st.Endpoints {
      if e.Stats == nil {
         continue // no traffic in the window
      }
      throughputByEndpoint[e.ID] = e.Stats.P50Throughput
   }

   // GPQA Diamond per provider. A provider can have several scored
   // endpoints (e.g. different quantizations); every score is kept, and
   // all of the provider's endpoint ids are collected for the tps join.
   gpqaByProvider := make(map[string]providerGPQA)
   for _, s := range st.Scores {
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
