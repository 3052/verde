// main.go
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

// assignCombined fills in s.Combined for every score.
//
// Each dimension — intelligence, median p50 tps, and provider count — is
// normalized over its interquartile range (Q1 to Q3), then clipped to
// [0, 1] and averaged with equal weight, scaled to 0-100:
//
//   X' = clamp((X - Q1) / (Q3 - Q1), 0, 1)
//   combined = (I' + T' + P') / 3 * 100
//
// The IQR is set by the middle 50% of candidates, so one extreme model
// cannot stretch the scale (min-max's failure mode). Clipping treats
// everything above Q3 as "fully good" and below Q1 as "fully bad" on that
// dimension. All three dimensions are treated identically, and values are
// continuous, so ties effectively disappear.
func assignCombined(scored []*score) {
   if len(scored) == 0 {
      return
   }

   intels := make([]float64, len(scored))
   tps := make([]float64, len(scored))
   counts := make([]float64, len(scored))
   for i, s := range scored {
      intels[i] = s.Intelligence
      tps[i] = s.MedianP50
      counts[i] = float64(len(s.Providers))
   }
   slices.Sort(intels)
   slices.Sort(tps)
   slices.Sort(counts)

   iQ1, iQ3 := percentile(intels, 25), percentile(intels, 75)
   tQ1, tQ3 := percentile(tps, 25), percentile(tps, 75)
   cQ1, cQ3 := percentile(counts, 25), percentile(counts, 75)

   for _, s := range scored {
      in := normIQR(s.Intelligence, iQ1, iQ3)
      tn := normIQR(s.MedianP50, tQ1, tQ3)
      pn := normIQR(float64(len(s.Providers)), cQ1, cQ3)
      s.Combined = (in + tn + pn) / 3 * 100
   }
}

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

// normIQR maps v into [0, 1] using the interquartile range [q1, q3], with
// values below q1 clamped to 0 and above q3 clamped to 1. If q1 == q3
// (e.g. too few candidates), every model gets the neutral 0.5.
func normIQR(v, q1, q3 float64) float64 {
   if q3 <= q1 {
      return 0.5
   }
   v = (v - q1) / (q3 - q1)
   if v < 0 {
      return 0
   }
   if v > 1 {
      return 1
   }
   return v
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
   fmt.Fprintf(os.Stderr,
      "combined = (I' + T' + P') / 3 * 100\n"+
         "  I' = clamp((I - Q1) / (Q3 - Q1), 0, 1)\n"+
         "  T' = clamp((T - Q1) / (Q3 - Q1), 0, 1)\n"+
         "  P' = clamp((P - Q1) / (Q3 - Q1), 0, 1)\n"+
         "  I = intelligence, T = median p50 tps, P = provider count\n")

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

   return nil
}

// main.go
