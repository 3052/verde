// api.go marker - preserve

package main

import (
   "encoding/json"
   "fmt"
   "net/http"
   "net/url"
)

// ---------------------------------------------------------------------------
// Endpoints
// ---------------------------------------------------------------------------

// One request for the whole catalog (confirmed by capture).
const catalogURL = "https://openrouter.ai/api/frontend/v1/models/find?active=true"

// Per-permaslug stats, one request each — no HTML, no escaped-JSON brace
// counting. Both endpoints live in the same /api/frontend/v1/stats
// namespace the model page's own queries use, are reachable with no cookie
// and no API key (verified live: identical payload from a clean client),
// and answer with access-control-allow-origin: *; benchmark-scores is
// additionally CDN-cached for an hour.
//
// The permaslug goes in the query string, so its "/" must be escaped to
// %2F, exactly as in the captured request. This is the opposite of the
// model page route, where PathEscape is wrong because the "/" is a path
// separator.
const scoresURLTemplate = "https://openrouter.ai/api/frontend/v1/stats/benchmark-scores?permaslug=%s"
const throughputURLTemplate = "https://openrouter.ai/api/frontend/v1/stats/throughput-comparison?permaslug=%s"

// fetchThroughput returns, per endpoint id, the tokens/sec of the most
// recent day that endpoint was measured on. The ids parameter the
// endpoint also accepts is ignored server-side (verified: byte-identical
// payload with and without it), so one request covers every endpoint of
// the model.
func fetchThroughput(c *http.Client, permaslug string) (map[string]float64, error) {
   body, err := httpGet(c, fmt.Sprintf(throughputURLTemplate, url.PathEscape(permaslug)))
   if err != nil {
      return nil, err
   }
   var resp throughputResponse
   if err := json.Unmarshal(body, &resp); err != nil {
      return nil, fmt.Errorf("decoding throughput stats: %w", err)
   }
   if resp.Data == nil {
      return nil, fmt.Errorf("no data array in throughput payload for %s", permaslug)
   }
   latest := make(map[string]string) // endpoint id -> day of the value kept
   throughput := make(map[string]float64)
   for _, p := range *resp.Data {
      for id, tps := range p.Y {
         if p.X >= latest[id] { // ties keep the later entry
            throughput[id] = tps
            latest[id] = p.X
         }
      }
   }
   return throughput, nil
}

// ---------------------------------------------------------------------------
// Catalog: single request -> open-weights candidates
// ---------------------------------------------------------------------------

func fetchCandidates(c *http.Client, minIntelligence float64) ([]candidate, int, error) {
   // The intelligence filter is applied server-side: the models page
   // appends min_intelligence_index to the find request (confirmed by
   // capture). Zero means no filter. NB: fetchURL, not url — the url
   // package is imported for PathEscape in the stats helpers below.
   fetchURL := catalogURL
   if minIntelligence > 0 {
      fetchURL = fmt.Sprintf("%s&min_intelligence_index=%g", catalogURL, minIntelligence)
   }
   body, err := httpGet(c, fetchURL)
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

      // The permaslug is the key for both stats requests — no derived
      // model page path is needed any more.
      cd := candidate{slug: m.Permaslug}
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

type findResponse struct {
   Data struct {
      Models     []catalogModel        `json:"models"`
      Benchmarks map[string]benchmarks `json:"benchmarks"`
   } `json:"data"`
}

// ---------------------------------------------------------------------------
// Scores: one per-permaslug request -> AutoExacto entries
// ---------------------------------------------------------------------------

// scoreEntry is one entry of the benchmark-scores array (confirmed by
// capture). endpoint_id is null for the auto-routing pseudo-provider and
// decodes to the empty string.
type scoreEntry struct {
   ProviderName  string  `json:"provider_name"`
   BenchmarkType string  `json:"benchmark_type"`
   Score         float64 `json:"score"`
   RunCount      int     `json:"run_count"`
   EndpointID    string  `json:"endpoint_id"`
}

// fetchScores returns one permaslug's AutoExacto score entries, all
// benchmark types. A payload with no entries at all is an ordinary error:
// the request is made once, and every failure (transport, HTTP status,
// decoding, missing scores) aborts the run.
func fetchScores(c *http.Client, permaslug string) ([]scoreEntry, error) {
   body, err := httpGet(c, fmt.Sprintf(scoresURLTemplate, url.PathEscape(permaslug)))
   if err != nil {
      return nil, err
   }
   var resp scoresResponse
   if err := json.Unmarshal(body, &resp); err != nil {
      return nil, fmt.Errorf("decoding benchmark scores: %w", err)
   }
   if len(resp.Data.Scores) == 0 {
      return nil, fmt.Errorf("no benchmark score entries for %s", permaslug)
   }
   return resp.Data.Scores, nil
}

// scoresResponse is the benchmark-scores payload. It carries every
// benchmark type for the permaslug (gpqa_diamond and
// tau_bench_verified_airline both arrive in one response); the caller
// filters.
type scoresResponse struct {
   Data struct {
      Scores []scoreEntry `json:"scores"`
   } `json:"data"`
}

// throughputPoint is one day of the series. x is the day, kept as a
// string so the latest point is picked by date rather than by payload
// order (the observed payload is ascending, but nothing documents that).
type throughputPoint struct {
   X string             `json:"x"`
   Y map[string]float64 `json:"y"`
}

// ---------------------------------------------------------------------------
// Throughput: one per-permaslug request -> tokens/sec per endpoint
// ---------------------------------------------------------------------------

// throughputResponse is the throughput-comparison payload: a daily
// series, one point per day, each point's y mapping endpoint id ->
// tokens/sec. Data is a pointer so that a payload without a data key at
// all (a shape change) is distinguishable from one carrying an empty
// series (a model with no traffic yet): the former is an error, the
// latter is not.
type throughputResponse struct {
   Data *[]throughputPoint `json:"data"`
}

// api.go marker - preserve
