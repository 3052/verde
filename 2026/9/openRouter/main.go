// main.go
package main

import (
   "cmp"
   "encoding/json"
   "flag"
   "fmt"
   "io"
   "net/http"
   "os"
   "slices"
   "strings"
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

// Per-provider p50 throughput, tokens/sec.

// MedianP50 is the median across providers of p50 throughput,
// tokens/sec.
// Combined is assigned after all scores are fetched: a 0-100 blend of
// intelligence, median p50 throughput, and provider count. Higher is
// better.

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

// modelPageSlug derives the model page path from a permaslug by
// stripping the trailing version date: "z-ai/glm-5.3-20260816" ->
// "z-ai/glm-5.3". A permaslug without a trailing 8-digit date is
// returned unchanged.
func modelPageSlug(permaslug string) string {
   i := strings.LastIndex(permaslug, "-")
   if i < 0 {
      return permaslug
   }
   date := permaslug[i+1:]
   if len(date) != 8 {
      return permaslug
   }
   for _, r := range date {
      if r < '0' || r > '9' {
         return permaslug
      }
   }
   return permaslug[:i]
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

// normIQR maps v into [0, 1] using the interquartile range [q1, q3], with
// values below q1 clamped to 0 and above q3 clamped to 1. If q1 == q3
// (e.g. too few candidates), every model gets the neutral 0.5.

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
   var rows []row
   for i, cd := range cands {
      fmt.Fprintf(os.Stderr, "[%d/%d] %s ", i+1, len(cands), cd.slug)
      rs, err := fetchRows(client, cd)
      if err != nil {
         fmt.Fprintf(os.Stderr, "error: %v\n", err)
         continue
      }
      fmt.Fprintf(os.Stderr, "aa: %.1f\n", cd.intelligence)
      rows = append(rows, rs...)
   }

   // --- 3. Combine intelligence, median p50 throughput, and provider
   // count into one score.

   // --- 4. Sort by combined score, descending.
   slices.SortFunc(rows, func(a, b row) int {
      return cmp.Compare(b.GPQA, a.GPQA)
   })

   fmt.Printf("sorted by: gpqa diamond, descending\n\n")

   for _, r := range rows {
      fmt.Printf("model: %s\n", r.Model)
      fmt.Printf("provider: %s\n", r.Provider)
      fmt.Printf("gpqa diamond: %.1f%%\n", r.GPQA*100)
      fmt.Printf("p50 tps: %.1f\n", r.P50)
      fmt.Println()
   }

   return nil
}

type benchmarks struct {
   AA *struct {
      IntelligenceIndex float64 `json:"intelligence_index"`
   } `json:"aa"`
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
// Catalog: single request -> open-weights candidates
// ---------------------------------------------------------------------------

func fetchCandidates(c *http.Client, minIntelligence float64) ([]candidate, int, error) {
   body, err := httpGet(c, catalogURL)
   if err != nil {
      return nil, 0, err
   }
   var resp findResponse
   if err := json.Unmarshal(body, &resp); err != nil {
      return nil, 0, fmt.Errorf("decoding catalog: %w", err)
   }

   total := len(resp.Data.Models)
   seen := make(map[string]bool)
   var cands []candidate
   for _, m := range resp.Data.Models {
      // Open weights = weights published on Hugging Face.
      if m.Permaslug == "" || m.HfSlug == "" {
         continue
      }
      // The catalog can contain duplicate entries per permaslug.
      if seen[m.Permaslug] {
         continue
      }
      seen[m.Permaslug] = true

      cd := candidate{slug: m.Permaslug, name: m.Name}
      // The page lives at the model slug, not the permaslug.
      cd.pageSlug = modelPageSlug(m.Permaslug)
      if b, ok := resp.Data.Benchmarks[m.Permaslug]; ok && b.AA != nil {
         cd.intelligence = b.AA.IntelligenceIndex
      }
      if minIntelligence > 0 && cd.intelligence < minIntelligence {
         continue
      }
      cands = append(cands, cd)
   }
   return cands, total, nil
}

type catalogModel struct {
   Permaslug string `json:"permaslug"`
   Name      string `json:"name"`
   HfSlug    string `json:"hf_slug"` // non-empty == open weights
}

// ---------------------------------------------------------------------------
// Catalog types (from the models/find payload)
// ---------------------------------------------------------------------------

type findResponse struct {
   Data struct {
      Models     []catalogModel        `json:"models"`
      Benchmarks map[string]benchmarks `json:"benchmarks"`
   } `json:"data"`
}

// main.go
