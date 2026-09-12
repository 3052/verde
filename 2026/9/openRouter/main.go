// main.go marker - preserve

package main

import (
   "cmp"
   "flag"
   "fmt"
   "net/http"
   "os"
   "slices"
   "strings"
   "time"
)

func main() {
   minIntelligence := flag.Float64("i", 0,
      "drop candidates below this AA intelligence index (0 = no filter)")
   minGPQA := flag.Float64("g", 0,
      "drop providers below this GPQA Diamond score (0..1, 0 = no filter)")
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
   // request is reported and skipped — no retries.
   var rows []row
   for i, cd := range cands {
      fmt.Fprintf(os.Stderr, "[%d/%d] %s ", i+1, len(cands), cd.slug)
      rs, err := fetchRows(client, &cd)
      if err != nil {
         fmt.Fprintf(os.Stderr, "error: %v\n", err)
         continue
      }
      fmt.Fprintf(os.Stderr, "aa: %.1f\n", cd.intelligence)
      rows = append(rows, rs...)
   }

   // --- 3. Apply the -g floor: drop providers below minGPQA.
   rows = slices.DeleteFunc(rows, func(r row) bool {
      return r.GPQA < minGPQA
   })

   // --- 4. Sort by GPQA Diamond score, descending.
   slices.SortFunc(rows, func(a, b row) int {
      return cmp.Compare(b.GPQA, a.GPQA)
   })

   fmt.Printf("sorted by: gpqa diamond, descending\n\n")

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
   name         string
   intelligence float64
   // pageSlug is the model page path for this candidate (permaslug
   // minus the version date).
   pageSlug string
}

// ---------------------------------------------------------------------------
// Rows: one model page -> one row per provider
// ---------------------------------------------------------------------------

// providerGPQA aggregates one provider's GPQA entries: the kept score
// plus every endpoint id the provider was scored on, for the tps join.
type providerGPQA struct {
   GPQA        float64
   RunCount    int
   EndpointIDs []string
}

// ---------------------------------------------------------------------------
// Working types
// ---------------------------------------------------------------------------

// row is one model + one provider: the flat unit of output.
type row struct {
   Model    string
   Name     string
   Provider string
   // GPQA is the provider's AutoExacto GPQA Diamond score, 0..1.
   GPQA float64
   // Throughput is that provider's median p50 throughput, tokens/sec.
   Throughput float64
}

// fetchRows returns one row per provider: provider, GPQA Diamond score,
// and median p50 throughput, all from a single model page request. A row
// is emitted only when the provider has both a GPQA score and a tps
// measurement.
func fetchRows(c *http.Client, cd *candidate) ([]row, error) {
   st, err := fetchPageState(c, cd.pageSlug)
   if err != nil {
      return nil, err
   }

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
   // endpoints (e.g. different quantizations); the entry with the most
   // runs is kept as the most reliable, and all of the provider's
   // endpoint ids are collected for the tps join.
   gpqaByProvider := make(map[string]providerGPQA)
   for _, s := range st.Scores {
      if s.BenchmarkType != "gpqa_diamond" {
         continue
      }
      g, seen := gpqaByProvider[s.ProviderName]
      if !seen || s.RunCount > g.RunCount {
         g.GPQA, g.RunCount = s.Score, s.RunCount
      }
      if s.EndpointID != "" && !slices.Contains(g.EndpointIDs, s.EndpointID) {
         g.EndpointIDs = append(g.EndpointIDs, s.EndpointID)
      }
      gpqaByProvider[s.ProviderName] = g
   }
   if len(gpqaByProvider) == 0 {
      return nil, fmt.Errorf("no gpqa_diamond scores in page payload")
   }

   var rows []row
   for provider, g := range gpqaByProvider {
      // One p50 throughput value per provider: all of its endpoints'
      // p50 throughputs are collected and reduced to one median below.
      var throughputs []float64
      for _, id := range g.EndpointIDs {
         if tp, ok := throughputByEndpoint[id]; ok {
            throughputs = append(throughputs, tp)
         }
      }
      if len(throughputs) == 0 {
         continue // no throughput for this provider -> no row
      }
      slices.Sort(throughputs)
      rows = append(rows, row{
         Model:      cd.slug,
         Name:       cd.name,
         Provider:   provider,
         GPQA:       g.GPQA,
         Throughput: percentile(throughputs, 50),
      })
   }
   return rows, nil
}

// main.go marker - preserve
