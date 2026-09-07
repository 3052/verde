// tp-rank: rank open-weights OpenRouter models by a combined
// intelligence × throughput × provider-count score, aggregated by median
// across providers.
//
//   go run . [-i 40]
package main

import (
   "cmp"
   "flag"
   "fmt"
   "math"
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

   fmt.Fprintf(os.Stderr,
      "filtering providers to known quantization of 8 bits or more\n")

   // --- 2. Providers: sequential, one request per candidate. A failed
   // request is reported and skipped — no retries.
   var scored []*score
   for i, cd := range cands {
      s := fetchScore(client, cd)
      fmt.Fprintf(os.Stderr, "[%d/%d] %s ", i+1, len(cands), cd.slug)
      if s.Error != "" {
         fmt.Fprintf(os.Stderr, "error: %s\n", s.Error)
         continue
      }
      fmt.Fprintf(os.Stderr, "\n")
      scored = append(scored, &s)
   }

   // --- 3. Combine intelligence, median p50 throughput, and provider
   // count into one score.
   assignCombined(scored)
   fmt.Fprintf(os.Stderr, "combined = mean(I', T', P') * 100, each min-max normalized\n"+
      "  I' = intelligence\n"+
      "  T' = log10(1 + median p50 tps)\n"+
      "  P' = provider count\n")

   // --- 4. Sort by combined score, descending.
   slices.SortFunc(scored, func(a, b *score) int {
      return cmp.Compare(b.Combined, a.Combined)
   })

   for _, s := range scored {
      fmt.Printf("model: %s\n", s.Model)
      fmt.Printf("intelligence: %.1f\n", s.Intelligence)
      fmt.Printf("median p50 tps: %.1f\n", s.MedianP50)
      fmt.Printf("providers: %d\n", len(s.Providers))
      fmt.Printf("combined: %.2f\n", s.Combined)
      fmt.Println()
   }
   fmt.Fprintf(os.Stderr, "ranked %d of %d candidates\n", len(scored), len(cands))

   return nil
}

// assignCombined fills in s.Combined for every score.
//
// Three dimensions — intelligence, speed, and provider count — are each
// min-max normalized to [0, 1] and averaged with equal weight, scaled to
// 0-100:
//
//   I' = (I - Imin) / (Imax - Imin)
//   L  = log10(1 + T)                      // log so one fast outlier
//   T' = (L - Lmin) / (Lmax - Lmin)        // doesn't crush the rest
//   P' = (P - Pmin) / (Pmax - Pmin)        // more providers is better
//   combined = (I' + T' + P') / 3 * 100
//
// Each dimension spans exactly [0, 1], so all three count equally. Values
// are continuous, so ties (common with percentile ranks, which live on a
// discrete grid) effectively disappear.
func assignCombined(scored []*score) {
   if len(scored) == 0 {
      return
   }

   intels := make([]float64, len(scored))
   logs := make([]float64, len(scored))
   counts := make([]float64, len(scored))
   for i, s := range scored {
      intels[i] = s.Intelligence
      logs[i] = math.Log10(1 + s.MedianP50)
      counts[i] = float64(len(s.Providers))
   }

   iMin, iMax := minMax(intels)
   lMin, lMax := minMax(logs)
   cMin, cMax := minMax(counts)

   for _, s := range scored {
      in := norm(s.Intelligence, iMin, iMax)
      tn := norm(math.Log10(1+s.MedianP50), lMin, lMax)
      pn := norm(float64(len(s.Providers)), cMin, cMax)
      s.Combined = (in + tn + pn) / 3 * 100
   }
}

// norm maps v into [0, 1] within [lo, hi]. If all values are equal, every
// model gets the neutral 0.5.
func norm(v, lo, hi float64) float64 {
   if hi <= lo {
      return 0.5
   }
   return (v - lo) / (hi - lo)
}

func minMax(xs []float64) (float64, float64) {
   lo, hi := xs[0], xs[0]
   for _, x := range xs[1:] {
      if x < lo {
         lo = x
      }
      if x > hi {
         hi = x
      }
   }
   return lo, hi
}

// main.go
