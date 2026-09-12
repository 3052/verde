// api.go
package main

import (
   "encoding/json"
   "fmt"
   "net/http"
   "slices"
   "strings"
)

// ---------------------------------------------------------------------------
// Endpoints
// ---------------------------------------------------------------------------

// One request for the whole catalog (confirmed by capture).
const catalogURL = "https://openrouter.ai/api/frontend/v1/models/find?active=true"

// Per-model request for provider throughput stats (confirmed by capture).
// Response is {"data": [ {provider_name, stats: {p50_throughput, ...}}, ... ]}.

// Model page: one request per candidate carries everything — the
// AutoExacto scores array and the per-endpoint stats array are both
// embedded in the page payload as escaped JSON inside a script tag
// (confirmed by capture). There are no JSON endpoints for either.
// The page path is the model slug, NOT the permaslug — the permaslug
// page route errors, and escaping the slug is wrong too because
// PathEscape turns the "/" into "%2F" (verified live).
const pageURLTemplate = "https://openrouter.ai/%s"

// Marker for the start of one embedded score object, in its escaped
// form: {\"provider_name\":...
const scoreMarker = `{\"provider_name\"`

// Marker inside every embedded stats object, in its escaped form:
// \"p50_throughput\":...
const statsMarker = `\"p50_throughput\"`

// fetchGPQA returns one GPQA Diamond aggregate per provider from the
// scores array embedded in the model page payload. A provider can have
// several endpoints with separate scores (e.g. Baseten fp8 and bf16);
// the entry with the most runs is kept as the most reliable, and all of
// the provider's endpoint ids are collected for the tps join.
func fetchGPQA(html string) (map[string]providerGPQA, error) {
   out := make(map[string]providerGPQA)
   i := strings.Index(html, scoreMarker)
   for i >= 0 {
      // Brace-count to the end of the object; braces are not escaped,
      // only quotes are.
      end := findObjectEnd(html, i)
      if end < 0 {
         break // unterminated -> malformed payload, stop
      }
      esc := html[i : end+1]

      next := strings.Index(html[end+1:], scoreMarker)
      if next < 0 {
         i = -1
      } else {
         i = end + 1 + next
      }

      // The escaped body is the content of a JSON string: wrap it in
      // quotes and decode twice.
      var decoded string
      if err := json.Unmarshal([]byte("\""+esc+"\""), &decoded); err != nil {
         continue
      }
      var s autoExactoScore
      if err := json.Unmarshal([]byte(decoded), &s); err != nil {
         continue
      }
      if s.BenchmarkType != "gpqa_diamond" {
         continue
      }
      g, seen := out[s.ProviderName]
      // The entry with the most runs is kept as the most reliable.
      if !seen || s.RunCount > g.RunCount {
         g.GPQA = s.Score
         g.RunCount = s.RunCount
      }
      if s.EndpointID != "" && !slices.Contains(g.EndpointIDs, s.EndpointID) {
         g.EndpointIDs = append(g.EndpointIDs, s.EndpointID)
      }
      out[s.ProviderName] = g
   }
   if len(out) == 0 {
      return nil, fmt.Errorf("no gpqa_diamond scores in page payload")
   }
   return out, nil
}

// fetchP50s returns p50 throughput per endpoint id from the stats array
// embedded in the model page payload.
func fetchP50s(html string) map[string]float64 {
   out := make(map[string]float64)
   i := strings.Index(html, statsMarker)
   for i >= 0 {
      start := findObjectStart(html, i)
      if start < 0 {
         break
      }
      end := findObjectEnd(html, start)
      if end < 0 {
         break // unterminated -> malformed payload, stop
      }
      esc := html[start : end+1]

      next := strings.Index(html[end+1:], statsMarker)
      if next < 0 {
         i = -1
      } else {
         i = end + 1 + next
      }

      // The escaped body is the content of a JSON string: wrap it in
      // quotes and decode twice.
      var decoded string
      if err := json.Unmarshal([]byte("\""+esc+"\""), &decoded); err != nil {
         continue
      }
      var st pageEndpointStats
      if err := json.Unmarshal([]byte(decoded), &st); err != nil {
         continue
      }
      if st.ID != "" {
         out[st.ID] = st.P50Throughput
      }
   }
   return out
}

// ---------------------------------------------------------------------------
// Math / HTTP helpers
// ---------------------------------------------------------------------------

// quantBits returns the bit width encoded in a quantization label
// (e.g. "fp8" -> 8, "bf16" -> 16). It returns -1 when the label carries
// no width, such as "unknown".

// findObjectEnd returns the index of the '}' closing the object that
// opens at start, or -1. Braces are not escaped in the payload, only
// quotes are.
func findObjectEnd(html string, start int) int {
   depth := 0
   for j := start; j < len(html); j++ {
      switch html[j] {
      case '{':
         depth++
      case '}':
         depth--
         if depth == 0 {
            return j
         }
      }
   }
   return -1
}

// findObjectStart returns the index of the '{' opening the object that
// contains position i, or -1. Scanning backwards, a '}' raises the depth
// and a '{' at depth zero is the opening brace.
func findObjectStart(html string, i int) int {
   depth := 0
   for j := i; j >= 0; j-- {
      switch html[j] {
      case '}':
         depth++
      case '{':
         if depth == 0 {
            return j
         }
         depth--
      }
   }
   return -1
}

// autoExactoScore is one entry of the scores array embedded in the model
// page payload (confirmed by capture).
type autoExactoScore struct {
   ProviderName  string  `json:"provider_name"`
   BenchmarkType string  `json:"benchmark_type"`
   Score         float64 `json:"score"`
   RunCount      int     `json:"run_count"`
   EndpointID    string  `json:"endpoint_id"` // null -> empty (auto-routing)
   DisplayName   string  `json:"display_name"`
}

// pageEndpointStats is one entry of the per-endpoint stats array embedded
// in the model page payload (confirmed by capture). The window is short
// (window_minutes: 30) and values are integers; quantization is not part
// of this payload.
type pageEndpointStats struct {
   ID            string  `json:"endpoint_id"`
   P50Throughput float64 `json:"p50_throughput"`
}

// providerGPQA aggregates one provider's AutoExacto GPQA entries: the
// kept score plus every endpoint id the provider was scored on, for the
// tps join.
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
   // P50 is that provider's median p50 throughput, tokens/sec.
   P50 float64
}

// ---------------------------------------------------------------------------
// Stats types (from the stats/endpoint payload, confirmed by capture)
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Providers: one request per candidate, sequential, no retry
// ---------------------------------------------------------------------------

// fetchRows returns one row per provider: provider, GPQA Diamond score,
// and median p50 throughput, all from a single model page request. A row
// is emitted only when the provider has both a GPQA score and a tps
// measurement.
func fetchRows(c *http.Client, cd candidate) ([]row, error) {
   body, err := httpGet(c, fmt.Sprintf(pageURLTemplate, cd.pageSlug))
   if err != nil {
      return nil, err
   }
   html := string(body)

   gpqaByProvider, err := fetchGPQA(html)
   if err != nil {
      return nil, err
   }
   p50ByEndpoint := fetchP50s(html)

   var rows []row
   for provider, g := range gpqaByProvider {
      // One p50 throughput value per provider.
      // A provider can appear on several endpoints (e.g. different
      // quantizations), so all of its p50s are collected here and reduced
      // to one median below.
      var p50s []float64
      for _, id := range g.EndpointIDs {
         if p50, ok := p50ByEndpoint[id]; ok {
            p50s = append(p50s, p50)
         }
      }
      if len(p50s) == 0 {
         continue // no throughput for this provider -> no row
      }
      slices.Sort(p50s)
      rows = append(rows, row{
         Model:    cd.slug,
         Name:     cd.name,
         Provider: provider,
         GPQA:     g.GPQA,
         P50:      percentile(p50s, 50),
      })
   }
   return rows, nil
}

// api.go
