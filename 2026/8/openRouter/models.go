package main

import (
   "encoding/json"
   "io"
   "net/http"
   "slices"
   "sort"
   "strconv"
)

// parsePricePerMillion converts a "$/token" string to "$/M tokens".
// Returns 0 for missing/zero/invalid (treated as "no value").
func parsePricePerMillion(s string) float64 {
   if p, err := strconv.ParseFloat(s, 64); err == nil && p > 0 {
      return p * 1e6
   }
   return 0
}

// ============================================================================
// Filter & Sort
// ============================================================================

// sortValue returns (value, present) for the requested sort key.
// "present" is true only when the value is > 0.
func sortValue(r ModelData, key string) (float64, bool) {
   switch key {
   case "elo":
      return float64(r.Elo), r.Elo > 0
   case "intelligence":
      return r.Intelligence, r.Intelligence > 0
   case "coding":
      return r.Coding, r.Coding > 0
   case "agentic":
      return r.Agentic, r.Agentic > 0
   }
   return 0, false
}

type AABenchmark struct {
   IntelligenceIndex float64 `json:"intelligence_index"`
   CodingIndex       float64 `json:"coding_index"`
   AgenticIndex      float64 `json:"agentic_index"`
}

// ============================================================================
// API response types (only the fields we read)
// ============================================================================

type APIResponse struct {
   Data struct {
      Models     []Model              `json:"models"`
      Benchmarks map[string]Benchmark `json:"benchmarks"`
   } `json:"data"`
}

// ============================================================================
// Fetch & Parse
// ============================================================================

func FetchAndParse(url string) (*APIResponse, error) {
   resp, err := http.Get(url)
   if err != nil {
      return nil, err
   }
   defer resp.Body.Close()
   body, err := io.ReadAll(resp.Body)
   if err != nil {
      return nil, err
   }
   var apiResp APIResponse
   if err := json.Unmarshal(body, &apiResp); err != nil {
      return nil, err
   }
   return &apiResp, nil
}

type Benchmark struct {
   AA *AABenchmark `json:"aa"`
   DA *DABenchmark `json:"da"`
}

type DABenchmark struct {
   MaxElo int `json:"max_elo"`
}

type Endpoint struct {
   Pricing Pricing `json:"pricing"`
}

type Model struct {
   Permaslug       string   `json:"permaslug"`
   Name            string   `json:"name"`
   CreatedAt       string   `json:"created_at"`
   ContextLength   int64    `json:"context_length"`
   HfSlug          string   `json:"hf_slug"`
   InputModalities []string `json:"input_modalities"`
   Endpoint        Endpoint `json:"endpoint"`
}

// ============================================================================
// Flattened row used downstream
// ============================================================================

type ModelData struct {
   Slug           string
   Name           string
   CreatedAt      string
   ContextLength  int64
   HfSlug         string
   HasImage       bool
   Elo            int
   Intelligence   float64
   Coding         float64
   Agentic        float64
   InputPrice     float64 // $/M tokens
   OutputPrice    float64 // $/M tokens
   CacheReadPrice float64 // $/M tokens
}

func BuildModelData(apiResp *APIResponse) []ModelData {
   var rows []ModelData
   for _, m := range apiResp.Data.Models {
      if m.Permaslug == "" {
         continue
      }
      row := ModelData{
         Slug:           m.Permaslug,
         Name:           m.Name,
         CreatedAt:      m.CreatedAt,
         ContextLength:  m.ContextLength,
         HfSlug:         m.HfSlug,
         HasImage:       slices.Contains(m.InputModalities, "image"),
         InputPrice:     parsePricePerMillion(m.Endpoint.Pricing.Prompt),
         OutputPrice:    parsePricePerMillion(m.Endpoint.Pricing.Completion),
         CacheReadPrice: parsePricePerMillion(m.Endpoint.Pricing.InputCacheRead),
      }
      if b, ok := apiResp.Data.Benchmarks[m.Permaslug]; ok {
         if b.AA != nil {
            row.Intelligence = b.AA.IntelligenceIndex
            row.Coding = b.AA.CodingIndex
            row.Agentic = b.AA.AgenticIndex
         }
         if b.DA != nil {
            row.Elo = b.DA.MaxElo
         }
      }
      rows = append(rows, row)
   }
   return rows
}

func FilterAndSort(rows []ModelData, key string) []ModelData {
   var results []ModelData
   for _, r := range rows {
      // Pricing is always required.
      if r.InputPrice == 0 && r.OutputPrice == 0 {
         continue
      }
      // The sort field must have a value.
      if _, ok := sortValue(r, key); !ok {
         continue
      }
      results = append(results, r)
   }

   // Sort descending by the chosen key.
   sort.Slice(results, func(i, j int) bool {
      vi, _ := sortValue(results[i], key)
      vj, _ := sortValue(results[j], key)
      return vi > vj
   })

   return results
}

// Pricing values are strings like "0.0000025" (USD per token).
type Pricing struct {
   Prompt         string `json:"prompt"`
   Completion     string `json:"completion"`
   InputCacheRead string `json:"input_cache_read"`
}
