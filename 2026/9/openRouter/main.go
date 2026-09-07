// tp-rank: rank open-weights OpenRouter models by per-provider throughput
// percentiles, aggregated by median across providers.
//
//   go run . [-i 40]
package main

import (
   "cmp"
   "flag"
   "fmt"
   "net/http"
   "os"
   "slices"
   "time"
)

func main() {
   minIntelligence := flag.Float64("i", 0,
      "drop candidates below this AA intelligence index (0 = no filter)")
   flag.Parse()

   // No flags -> do nothing.
   if flag.NFlag() == 0 {
      flag.Usage()
      return
   }

   if err := run(*minIntelligence); err != nil {
      fmt.Fprintf(os.Stderr, "%v\n", err)
      os.Exit(1)
   }
}

func run(minIntelligence float64) error {
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
   var scored []*score
   for i, cd := range cands {
      s := fetchScore(client, cd)
      fmt.Fprintf(os.Stderr, "[%d/%d] %-50s ", i+1, len(cands), cd.slug)
      if s.Error != "" {
         fmt.Fprintf(os.Stderr, "error: %s\n", s.Error)
         continue
      }
      fmt.Fprintf(os.Stderr, "%d providers, median p50 %.1f tps\n",
         len(s.Providers), s.Medians[0])
      scored = append(scored, &s)
   }

   // --- 3. Sort by intelligence, descending.
   slices.SortFunc(scored, func(a, b *score) int {
      return cmp.Compare(b.Intelligence, a.Intelligence)
   })
   fmt.Fprintf(os.Stderr, "sorted by intelligence, descending\n")

   for _, s := range scored {
      fmt.Printf("model: %s\n", s.Model)
      fmt.Printf("intelligence: %.1f\n", s.Intelligence)
      fmt.Printf("median p50 tps: %.1f\n", s.Medians[0])
      fmt.Printf("median p75 tps: %.1f\n", s.Medians[1])
      fmt.Printf("providers: %d\n", len(s.Providers))
      fmt.Println()
   }
   fmt.Fprintf(os.Stderr, "ranked %d of %d candidates\n", len(scored), len(cands))

   return nil
}

// main.go
