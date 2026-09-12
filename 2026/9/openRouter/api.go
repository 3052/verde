// api.go marker - preserve

package main

import (
   "bytes"
   "encoding/json"
   "fmt"
   "net/http"
   "slices"
)

// ---------------------------------------------------------------------------
// Endpoints
// ---------------------------------------------------------------------------

// One request for the whole catalog (confirmed by capture).
const catalogURL = "https://openrouter.ai/api/frontend/v1/models/find?active=true"

// Model page: one request per candidate carries everything — the
// AutoExacto scores and the per-endpoint stats are both embedded in the
// page as one escaped JSON blob inside a script tag (confirmed by
// capture). There are no JSON endpoints for either. The page path is the
// model slug, NOT the permaslug — the permaslug page route errors, and
// escaping the slug is wrong too because PathEscape turns the "/" into
// "%2F" (verified live).
const pageURLTemplate = "https://openrouter.ai/%s"

// ---------------------------------------------------------------------------
// Page payload: extraction
// ---------------------------------------------------------------------------

// Marker for the embedded query state, in its escaped form. The payload
// is a JSON string inside a script tag, so quotes are escaped but braces
// are not — brace counting works, string contents carry no braces.
var prefix = []byte(`{\"state\"`)

// TopJSON returns the bracketed value that starts at the first occurrence
// of prefix.
func TopJSON(in []byte) ([]byte, bool) {
   i := bytes.Index(in, prefix)
   if i < 0 {
      return nil, false
   }
   depth := 0
   for j := i; j < len(in); j++ {
      switch in[j] {
      case '{':
         depth++
      case '}':
         if depth--; depth == 0 {
            return in[i : j+1], true
         }
      }
   }
   return nil, false
}

// endpointEntry is one entry of the endpoint-stats arrays (confirmed by
// capture).
type endpointEntry struct {
   ID    string         `json:"id"`
   Stats *endpointStats `json:"stats"` // null: no traffic in the window
}

// endpointStats is the rolling-window (window_minutes: 30) measurement
// block of one endpoint entry.
type endpointStats struct {
   P50Throughput float64 `json:"p50_throughput"`
}

// ---------------------------------------------------------------------------
// Page payload: types
// ---------------------------------------------------------------------------

// pageState is the typed subset of the dehydrated state of one model
// page.
type pageState struct {
   // Endpoints holds the entries of the endpointStats and
   // providerTableEndpointStats queries' data arrays — the same
   // array, twice.
   Endpoints []endpointEntry
   // Scores holds the entries of the benchmarkScores query's scores
   // array.
   Scores []scoreEntry
}

// fetchPageState fetches a model page and returns the typed subset of its
// embedded query state: the AutoExacto scores and the per-endpoint
// throughput stats.
func fetchPageState(c *http.Client, pageSlug string) (*pageState, error) {
   body, err := httpGet(c, fmt.Sprintf(pageURLTemplate, pageSlug))
   if err != nil {
      return nil, err
   }
   esc, ok := TopJSON(body)
   if !ok {
      return nil, fmt.Errorf("no embedded query state in page payload")
   }
   // esc is the body of a JSON string (quotes escaped): quote it and
   // decode the escapes once.
   quoted := make([]byte, 0, len(esc)+2)
   quoted = append(quoted, '"')
   quoted = append(quoted, esc...)
   quoted = append(quoted, '"')
   var raw string
   if err := json.Unmarshal(quoted, &raw); err != nil {
      return nil, fmt.Errorf("unescaping page payload: %w", err)
   }

   // First pass: decode the shape only. Each query keeps its data raw —
   // the payload type is not known until the key is read.
   var dehydrated struct {
      State struct {
         Queries []rawQuery `json:"queries"`
      } `json:"state"`
   }
   if err := json.Unmarshal([]byte(raw), &dehydrated); err != nil {
      return nil, fmt.Errorf("decoding page payload: %w", err)
   }

   // Second pass: per query key, one proper unmarshal of its data.
   st := &pageState{}
   for _, q := range dehydrated.State.Queries {
      if len(q.State.Data) == 0 {
         continue // no data (e.g. a pending query)
      }
      switch {
      case q.hasKey("endpointStats"), q.hasKey("providerTableEndpointStats"):
         var eps []endpointEntry
         if err := json.Unmarshal(q.State.Data, &eps); err != nil {
            return nil, fmt.Errorf("decoding endpoint stats: %w", err)
         }
         st.Endpoints = append(st.Endpoints, eps...)
      case q.hasKey("benchmarkScores"):
         var d struct {
            Scores []scoreEntry `json:"scores"`
         }
         if err := json.Unmarshal(q.State.Data, &d); err != nil {
            return nil, fmt.Errorf("decoding benchmark scores: %w", err)
         }
         st.Scores = append(st.Scores, d.Scores...)
      }
   }
   return st, nil
}

// providerGPQA aggregates one provider's GPQA entries: the kept score
// plus every endpoint id the provider was scored on, for the tps join.
type providerGPQA struct {
   GPQA        float64
   RunCount    int
   EndpointIDs []string
}

// rawQuery is one entry of the dehydrated state's queries array. The
// query key is heterogeneous — ["model-page","endpointStats",{...}]:
// strings plus a trailing options object — and the payload type differs
// per key, so both stay raw until the key identifies the type.
type rawQuery struct {
   QueryKey []json.RawMessage `json:"queryKey"`
   State    struct {
      Data json.RawMessage `json:"data"`
   } `json:"state"`
}

// hasKey reports whether the query's key contains the string element s.
// Non-string key elements (the trailing options object) fail the string
// decode and are skipped.
func (q *rawQuery) hasKey(s string) bool {
   for _, k := range q.QueryKey {
      var v string
      if json.Unmarshal(k, &v) == nil && v == s {
         return true
      }
   }
   return false
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
// Rows: one model page -> one row per provider
// ---------------------------------------------------------------------------

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
   p50ByEndpoint := make(map[string]float64)
   for _, e := range st.Endpoints {
      if e.Stats == nil {
         continue // no traffic in the window
      }
      p50ByEndpoint[e.ID] = e.Stats.P50Throughput
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
      // p50s are collected and reduced to one median below.
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

// scoreEntry is one entry of the AutoExacto scores array (confirmed by
// capture). endpoint_id is null for the auto-routing pseudo-provider
// and decodes to the empty string.
type scoreEntry struct {
   ProviderName  string  `json:"provider_name"`
   BenchmarkType string  `json:"benchmark_type"`
   Score         float64 `json:"score"`
   RunCount      int     `json:"run_count"`
   EndpointID    string  `json:"endpoint_id"`
}

// api.go marker - preserve
