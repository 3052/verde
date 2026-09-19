// api.go marker - preserve

package main

import (
   "encoding/json"
   "errors"
   "fmt"
   "net/http"
   "net/url"
   "strings"
)

// One request for the whole catalog. The context floor is appended in
// fetchCandidates rather than baked in here, so the logged floor and the
// sent parameter cannot drift apart.
const catalogURL = "https://openrouter.ai/api/frontend/v1/models/find?active=true"

// ---------------------------------------------------------------------------
// Endpoints
// ---------------------------------------------------------------------------

// minContext is the model-context floor in tokens, sent to the catalog as
// the `context` query parameter. Hardcoded for now; it is logged at the
// start of every run so the cutoff is never implicit.
//
// Verified live against this find endpoint (not only against the documented
// /api/v1/models, which documents the same parameter name): with
// context=2000000 the payload holds exactly the five models whose
// context_length is 2000000 and no others, so the parameter is honored and
// the bound is inclusive (>=). The comparison is against the model's own
// context_length, not the served endpoints' (a provider may serve less than
// the model advertises).
const minContext = 1000000

// Per-permaslug stats, one request each — no HTML, no escaped-JSON brace
// counting. Both endpoints live in the same /api/frontend/v1/stats
// namespace the model page's own queries use, are reachable with no cookie
// and no API key (verified live: identical payload from a clean client),
// and answer with access-control-allow-origin: *; both are CDN-cached for
// an hour.
//
// The permaslug goes in the query string, so its "/" must be escaped to
// %2F, exactly as in the captured request. This is the opposite of the
// model page route, where PathEscape is wrong because the "/" is a path
// separator.
const scoresURLTemplate = "https://openrouter.ai/api/frontend/v1/stats/benchmark-scores?permaslug=%s"
const throughputURLTemplate = "https://openrouter.ai/api/frontend/v1/stats/throughput-comparison?permaslug=%s"

// ---------------------------------------------------------------------------
// Absent data: 200 responses that mean "skip this candidate", not "stop"
// ---------------------------------------------------------------------------

// These stats endpoints answer 200 with an empty payload for a permaslug
// that has nothing recorded yet. Captured:
//
//   GET benchmark-scores?permaslug=tencent%2Fhy4-preview-20260827
//   -> {"data":{"scores":[],"lookback_days":32}}
//
// Over a whole catalog that is routine — a newly published model has no
// benchmark runs and no traffic — so it must not abort the run. These
// sentinels mark it, and skippable is the single place that decides which
// errors mean "drop this candidate" rather than "stop".
//
// The line each fetcher draws is between a server that says it has no data
// (an empty array -> sentinel) and a server that no longer answers in the
// documented shape (a missing or null data object -> ordinary error). A
// genuine API change therefore still aborts loudly, instead of quietly
// emptying the output.
//
// errNoThroughput covers a throughput series that yields no measurement at
// all. It is deliberately distinct from a series that does carry
// measurements but not for any of the endpoints GPQA was scored on: the
// first is "nothing recorded", the second is a join that came up empty, and
// the per-candidate log reports them differently.
var (
   errNoScores     = errors.New("no benchmark scores")
   errNoGPQA       = errors.New("no gpqa_diamond scores")
   errNoThroughput = errors.New("no throughput stats")
)

func fetchCandidates(c *http.Client) ([]candidate, catalogStats, error) {
   var st catalogStats
   // The context floor is applied server-side. NB: fetchURL, not url — the
   // url package is imported for PathEscape in the stats helpers below.
   fetchURL := fmt.Sprintf("%s&context=%d", catalogURL, minContext)
   body, err := httpGet(c, fetchURL)
   if err != nil {
      return nil, st, err
   }
   var resp findResponse
   if err := json.Unmarshal(body, &resp); err != nil {
      return nil, st, fmt.Errorf("decoding catalog: %w", err)
   }

   st.AfterContext = len(resp.Data.Models)
   seen := make(map[string]bool)
   var cands []candidate
   for _, m := range resp.Data.Models {
      // Open weights = weights published on Hugging Face. A closed model
      // carries hf_slug as null or as "" — both decode to the empty string,
      // so one check covers them (verified live on both spellings).
      if m.Permaslug == "" || m.HfSlug == "" {
         st.DroppedClosed++
         continue
      }
      // The catalog can contain duplicate entries per permaslug.
      if seen[m.Permaslug] {
         st.DroppedDuplicate++
         continue
      }
      seen[m.Permaslug] = true

      // The permaslug is the key for both stats requests.
      cands = append(cands, candidate{slug: m.Permaslug, contextLength: m.ContextLength})
   }
   return cands, st, nil
}

// fetchThroughput returns, per endpoint id, the tokens/sec of the most
// recent day that endpoint was measured on. The ids parameter the endpoint
// also accepts is ignored server-side (verified: byte-identical payload
// with and without it), so one request covers every endpoint of the model.
//
// A payload that yields no measurement at all — an empty series, or points
// whose y maps are all empty — returns errNoThroughput, which the caller
// treats as a skip rather than letting the candidate collapse silently to
// zero rows.
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
   if len(throughput) == 0 {
      return nil, errNoThroughput
   }
   return throughput, nil
}

// filterReport renders the filter chain: which filters ran, which side
// applied each, and how many models each step removed. The cutoff is
// printed from minContext, the same constant the request URL is built
// from, so the logged cutoff cannot drift from the sent one.
func filterReport(st catalogStats) string {
   data := &strings.Builder{}
   fmt.Fprintf(data, "catalog filters, in order:\n")
   fmt.Fprintf(data, "  1. context >= %d tokens   server-side, models/find `context` (model card)\n", minContext)
   fmt.Fprintf(data, "  2. open weights           client-side, hf_slug non-empty\n")
   fmt.Fprintf(data, "  3. distinct permaslug     client-side, first occurrence kept\n")
   fmt.Fprintf(data, "\n")
   fmt.Fprintf(data, "  %-33s %6d\n", "after filter 1 (context):", st.AfterContext)
   fmt.Fprintf(data, "  %-33s %6d\n", "dropped by filter 2 (no weights):", st.DroppedClosed)
   fmt.Fprintf(data, "  %-33s %6d\n", "dropped by filter 3 (duplicates):", st.DroppedDuplicate)
   fmt.Fprintf(data, "  %-33s %6d\n", "candidates:",
      st.AfterContext-st.DroppedClosed-st.DroppedDuplicate)
   return data.String()
}

// skippable reports whether err means "no data for this candidate" — the
// candidate is dropped and the run continues — rather than a real failure.
func skippable(err error) bool {
   return errors.Is(err, errNoScores) ||
      errors.Is(err, errNoGPQA) ||
      errors.Is(err, errNoThroughput)
}

type catalogModel struct {
   Permaslug string `json:"permaslug"`
   HfSlug    string `json:"hf_slug"` // empty == not open weights (null or "")
   // ContextLength is the model card's window — the field the context
   // query parameter filters on. Echoed per candidate in the progress block;
   // the model card value, not a per-endpoint limit, which can be lower.
   ContextLength int `json:"context_length"`
}

// ---------------------------------------------------------------------------
// Catalog: single request -> open-weights candidates
// ---------------------------------------------------------------------------

// catalogStats counts what the catalog filters removed, in the order they
// ran. It exists so the run log can name every filter and its effect
// instead of printing a bare survivor count that looks like the size of
// the whole catalog.
//
// AfterContext is what the server returned for the context filter — already
// filtered, NOT the whole catalog. The two dropped counts are the
// client-side steps, in code order (a model that is both closed and a
// duplicate counts under DroppedClosed, because that check runs first), so
// the three numbers always sum to the candidate list.
type catalogStats struct {
   AfterContext     int // models the server returned for context >= minContext
   DroppedClosed    int // of those, dropped for having no Hugging Face weights
   DroppedDuplicate int // of those, dropped for repeating an earlier permaslug
}

// findResponse carries only the fields this program reads. The payload also
// has a benchmarks map (AA intelligence index and friends) alongside models;
// it is neither decoded nor used, and unknown JSON fields are ignored, so
// nothing needs to name it here.
type findResponse struct {
   Data struct {
      Models []catalogModel `json:"models"`
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
// benchmark types. The request is made once, and every real failure
// (transport, HTTP status, decoding, a payload without a data object)
// aborts the run; a 200 that simply carries no scores returns errNoScores,
// which the caller treats as a skip.
func fetchScores(c *http.Client, permaslug string) ([]scoreEntry, error) {
   body, err := httpGet(c, fmt.Sprintf(scoresURLTemplate, url.PathEscape(permaslug)))
   if err != nil {
      return nil, err
   }
   var resp scoresResponse
   if err := json.Unmarshal(body, &resp); err != nil {
      return nil, fmt.Errorf("decoding benchmark scores: %w", err)
   }
   if resp.Data == nil {
      return nil, fmt.Errorf("no data object in benchmark scores payload for %s", permaslug)
   }
   if len(resp.Data.Scores) == 0 {
      return nil, errNoScores
   }
   return resp.Data.Scores, nil
}

// scoresResponse is the benchmark-scores payload. It carries every
// benchmark type for the permaslug (gpqa_diamond and
// tau_bench_verified_airline both arrive in one response); the caller
// filters.
//
// Data is a pointer so that the two empty-looking payloads stay
// distinguishable: no data object at all is a shape change and an ordinary
// error, while a data object with an empty scores array is the server
// reporting that it has nothing for this permaslug (the tencent case).
type scoresResponse struct {
   Data *struct {
      Scores []scoreEntry `json:"scores"`
   } `json:"data"`
}

// throughputPoint is one day of the series. x is the day, kept as a string
// so the latest point is picked by date rather than by payload order (the
// observed payload is ascending, but nothing documents that).
type throughputPoint struct {
   X string             `json:"x"`
   Y map[string]float64 `json:"y"`
}

// ---------------------------------------------------------------------------
// Throughput: one per-permaslug request -> tokens/sec per endpoint
// ---------------------------------------------------------------------------

// throughputResponse is the throughput-comparison payload: a daily series,
// one point per day, each point's y mapping endpoint id -> tokens/sec. Data
// is a pointer so that a payload without a data key at all (a shape change)
// is distinguishable from one carrying an empty series (a model with no
// traffic yet): the former is an error, the latter is not.
type throughputResponse struct {
   Data *[]throughputPoint `json:"data"`
}

// api.go marker - preserve
