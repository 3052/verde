package main

import (
   "flag"
   "fmt"
   "os"
)

func main() {
   openOnly := flag.Bool("open", false,
      "only show models with open weights")
   imageOnly := flag.Bool("image", false,
      "only show models that accept image input")
   export := flag.Bool("export", false,
      "fetch models and print them sorted by intelligence")
   flag.Parse()

   // No flags -> do nothing.
   if !*export {
      flag.Usage()
      return
   }

   // Fetch
   url := "https://openrouter.ai/api/frontend/v1/models/find?active=true"
   fmt.Fprintf(os.Stderr, "Fetching from %s ...\n", url)
   apiResp, err := FetchAndParse(url)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error: %v\n", err)
      os.Exit(1)
   }

   // Build model data
   rows := BuildModelData(apiResp)

   // Filter by open weights if specified
   if *openOnly {
      var filtered []ModelData
      for _, r := range rows {
         if r.HfSlug != "" {
            filtered = append(filtered, r)
         }
      }
      rows = filtered
   }

   // Filter by image input if specified
   if *imageOnly {
      var filtered []ModelData
      for _, r := range rows {
         if r.HasImage {
            filtered = append(filtered, r)
         }
      }
      rows = filtered
   }

   // Filter+sort by intelligence
   key := "intelligence"
   results := FilterAndSort(rows, key)

   out := os.Stdout
   fmt.Fprintf(out, "Sorted by: %s descending\n", key)
   if *openOnly {
      fmt.Fprintln(out, "Filter: open weights only")
   }
   if *imageOnly {
      fmt.Fprintln(out, "Filter: image input only")
   }

   if len(results) == 0 {
      fmt.Fprintln(out, "No models match the criteria")
      fmt.Fprintf(os.Stderr, "Printed 0 models\n")
      return
   }

   for _, r := range results {
      fmt.Fprintln(out)
      fmt.Fprintf(out, "Model: %s\n", r.Name)
      fmt.Fprintf(out, "Created: %s\n", r.CreatedAt)
      fmt.Fprintf(out, "Context length: %d tokens\n", r.ContextLength)
      if r.HfSlug != "" {
         fmt.Fprintf(out, "Model weights: %s\n", r.HfSlug)
      }
      if r.HasImage {
         fmt.Fprintln(out, "Image input: yes")
      }
      if r.Elo > 0 {
         fmt.Fprintf(out, "Arena ELO: %d\n", r.Elo)
      }
      fmt.Fprintf(out, "Intelligence: %.1f\n", r.Intelligence)
      fmt.Fprintf(out, "Coding: %.1f\n", r.Coding)
      fmt.Fprintf(out, "Agentic: %.1f\n", r.Agentic)
      fmt.Fprintf(out, "Input price: $%.2f / M tokens\n", r.InputPrice)
      fmt.Fprintf(out, "Output price: $%.2f / M tokens\n", r.OutputPrice)
      fmt.Fprintf(out, "Cache read price: $%.2f / M tokens\n", r.CacheReadPrice)
   }

   fmt.Fprintf(out, "\nRanked %d models (out of %d total)\n",
      len(results), len(rows))
   fmt.Fprintf(os.Stderr, "Printed %d models\n", len(results))
}
