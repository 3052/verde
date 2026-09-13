// api.go marker - preserve

package main

import (
   "bytes"
   "encoding/json"
   "fmt"
   "net/http"
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

// The payload is a JSON string inside a script tag, so quotes are
// escaped but braces are not — the queries array is located by brace
// counting over its object elements. Needle for the dehydrated state's
// queries array, in its escaped form. The state object carries a
// mutations array before queries (confirmed by capture:
// {\"state\":{\"mutations\":[],\"queries\":[…), so the needle anchors on
// the escaped queries key itself — the only queries array on the page
// (the query elements use queryKey and queryHash, never a queries key).
var prefix = []byte(`\"queries\":[`)

// TopJSON returns the bracketed queries array that follows the first
// occurrence of prefix: the first element's opening brace starts a brace
// count, and the first closing bracket in a gap between two
// brace-verified objects (where only commas and whitespace are
// grammatical) closes the array.
func TopJSON(in []byte) ([]byte, bool) {
   i := bytes.Index(in, prefix)
   if i < 0 {
      return nil, false
   }
   lo := i + len(prefix) - 1 // the array's opening bracket
   depth := 0
   for j := lo + 1; j < len(in); j++ {
      switch in[j] {
      case '{':
         depth++
      case '}':
         if depth--; depth == 0 {
            continue
         }
      case ']':
         if depth == 0 {
            return in[lo : j+1], true
         }
      }
   }
   return nil, false
}

// ---------------------------------------------------------------------------
// Catalog: single request -> open-weights candidates
// ---------------------------------------------------------------------------

func fetchCandidates(c *http.Client, minIntelligence float64) ([]candidate, int, error) {
   // The intelligence filter is applied server-side: the models page
   // appends min_intelligence_index to the find request (confirmed by
   // capture). Zero means no filter.
   url := catalogURL
   if minIntelligence > 0 {
      url = fmt.Sprintf("%s&min_intelligence_index=%g", catalogURL, minIntelligence)
   }
   body, err := httpGet(c, url)
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

      cd := candidate{slug: m.Permaslug}
      // The page lives at the model slug, not the permaslug.
      cd.pageSlug = modelPageSlug(m.Permaslug)
      // Display only — the filter was already applied server-side.
      if b, ok := resp.Data.Benchmarks[m.Permaslug]; ok && b.AA != nil {
         cd.intelligence = b.AA.IntelligenceIndex
      }
      cands = append(cands, cd)
   }
   return cands, total, nil
}

type benchmarks struct {
   AA *struct {
      IntelligenceIndex float64 `json:"intelligence_index"`
   } `json:"aa"`
}

type catalogModel struct {
   Permaslug string `json:"permaslug"`
   HfSlug    string `json:"hf_slug"` // non-empty == open weights
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
// Catalog types (from the models/find payload)
// ---------------------------------------------------------------------------

type findResponse struct {
   Data struct {
      Models     []catalogModel        `json:"models"`
      Benchmarks map[string]benchmarks `json:"benchmarks"`
   } `json:"data"`
}

// ---------------------------------------------------------------------------
// Page payload: types
// ---------------------------------------------------------------------------

// pageState is the typed subset of the dehydrated queries of one model
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

   // Each query keeps its data raw — the payload type is not known until
   // the key is read.
   var queries []rawQuery
   if err := json.Unmarshal([]byte(raw), &queries); err != nil {
      return nil, fmt.Errorf("decoding page payload: %w", err)
   }

   // Per query key, one proper unmarshal of its data.
   st := &pageState{}
   for _, q := range queries {
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

// rawQuery is one entry of the dehydrated queries array. The query key is
// heterogeneous — ["model-page","endpointStats",{...}]: strings plus a
// trailing options object — and the payload type differs per key, so both
// stay raw until the key identifies the type.
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
