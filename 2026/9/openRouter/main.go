// tp-rank: rank open-weights OpenRouter models by the median of
// per-provider p50 throughput.
//
//   go run . [-min-intelligence 40] [-json out.json]
package main

import (
   "encoding/json"
   "flag"
   "fmt"
   "net/http"
   "os"
   "sort"
   "text/tabwriter"
   "time"
)

func main() {
   var (
      minIntelligence = flag.Float64("min-intelligence", 0,
         "drop candidates below this AA intelligence index (0 = no filter)")
      outJSON = flag.String("json", "", "also write results to this file")
   )
   flag.Parse()

   // No flags -> do nothing.
   if flag.NFlag() == 0 {
      flag.Usage()
      return
   }

   if err := run(*minIntelligence, *outJSON); err != nil {
      fmt.Fprintf(os.Stderr, "%v\n", err)
      os.Exit(1)
   }
}

func run(minIntelligence float64, outJSON string) error {
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
   var scored []score
   for i, cd := range cands {
      s := fetchScore(client, cd)
      fmt.Fprintf(os.Stderr, "[%d/%d] %-50s ", i+1, len(cands), cd.slug)
      if s.Error != "" {
         fmt.Fprintf(os.Stderr, "error: %s\n", s.Error)
         continue
      }
      fmt.Fprintf(os.Stderr, "%d providers, median %.1f tps\n",
         len(s.Providers), s.MedianThroughput)
      if len(s.Providers) > 0 {
         scored = append(scored, s)
      }
   }

   // --- 3. Sort by median throughput, descending.
   sort.Slice(scored, func(i, j int) bool {
      return scored[i].MedianThroughput > scored[j].MedianThroughput
   })

   w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
   fmt.Fprintln(w, "MED TPS\tPROV\tINTEL\tMODEL")
   for _, s := range scored {
      fmt.Fprintf(w, "%.1f\t%d\t%.1f\t%s\n",
         s.MedianThroughput, len(s.Providers), s.Intelligence, s.Model)
   }
   w.Flush()
   fmt.Fprintf(os.Stderr, "ranked %d of %d candidates\n", len(scored), len(cands))

   if outJSON != "" {
      b, err := json.MarshalIndent(scored, "", "  ")
      if err == nil {
         err = os.WriteFile(outJSON, b, 0o644)
      }
      if err != nil {
         return fmt.Errorf("writing json: %w", err)
      }
   }
   return nil
}

// main.go
