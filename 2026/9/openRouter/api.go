package main

import (
   "encoding/json"
   "fmt"
   "io"
   "net/http"
   "net/url"
   "sort"
)

// ---------------------------------------------------------------------------
// Endpoints
// ---------------------------------------------------------------------------

// One request for the whole catalog (confirmed by capture).
const catalogURL = "https://openrouter.ai/api/frontend/v1/models/find?active=true"

// Per-model request for provider throughput stats (confirmed by capture).
// Response is {"data": [ {provider_name, stats: {p50_throughput, ...}}, ... ]}.
const statsURLTemplate = "https://openrouter.ai/api/frontend/v1/stats/endpoint" +
   "?latencyMetric=latency&perfWorkload=text_generation&variant=standard&permaslug=%s"

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

// ---------------------------------------------------------------------------
// Math / HTTP helpers
// ---------------------------------------------------------------------------

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

type benchmarks struct {
   AA *struct {
      IntelligenceIndex float64 `json:"intelligence_index"`
   } `json:"aa"`
}

// ---------------------------------------------------------------------------
// Working types
// ---------------------------------------------------------------------------

type candidate struct {
   slug         string
   name         string
   intelligence float64
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

type providerStat struct {
   Provider   string  `json:"provider"`
   Throughput float64 `json:"p50_throughput_tps"`
}

type score struct {
   Model            string         `json:"model"`
   Name             string         `json:"name"`
   Intelligence     float64        `json:"intelligence"`
   MedianThroughput float64        `json:"median_p50_throughput_tps"`
   Providers        []providerStat `json:"providers"`
   Error            string         `json:"error,omitempty"`
}

// ---------------------------------------------------------------------------
// Providers: one request per candidate, sequential, no retry
// ---------------------------------------------------------------------------

func fetchScore(c *http.Client, cd candidate) score {
   s := score{Model: cd.slug, Name: cd.name, Intelligence: cd.intelligence}

   u := fmt.Sprintf(statsURLTemplate, url.QueryEscape(cd.slug))
   body, err := httpGet(c, u)
   if err != nil {
      s.Error = err.Error()
      return s
   }

   var sr statsResponse
   if err := json.Unmarshal(body, &sr); err != nil {
      s.Error = "decode: " + err.Error()
      return s
   }

   var tps []float64
   for _, ep := range sr.Data {
      if ep.Stats == nil {
         continue // no stats -> no data -> excluded
      }
      s.Providers = append(s.Providers, providerStat{
         Provider:   ep.ProviderName,
         Throughput: ep.Stats.P50Throughput,
      })
      tps = append(tps, ep.Stats.P50Throughput)
   }
   sort.Float64s(tps)
   s.MedianThroughput = percentile(tps, 50)
   return s
}

type statDetails struct {
   P50Throughput float64 `json:"p50_throughput"` // tokens/sec
}

type statEndpoint struct {
   ProviderName string       `json:"provider_name"`
   Stats        *statDetails `json:"stats"`
}

// ---------------------------------------------------------------------------
// Stats types (from the stats/endpoint payload, confirmed by capture)
// ---------------------------------------------------------------------------

type statsResponse struct {
   Data []statEndpoint `json:"data"`
}

// api.go
