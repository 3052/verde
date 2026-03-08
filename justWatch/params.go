package justWatch

import (
   "net/url"
   "strings"
)

var params_to_delete = []struct {
   date  string
   key   string
   value string
}{
   {"2026-03-08", "searchReferral", ""},
   {"2026-03-07", "referrer", "JustWatch"},
   {"2026-03-04", "subId3", "justappsvod"},
   {"2026-02-26", "autoplay", "1"},
   {"2026-02-26", "searchReferral", "publisher"},
   {"2026-02-26", "source", "bing"},
   {"2026-02-26", "source", "search-feeds"},
   {"2026-02-26", "utm_campaign", "vod_feed"},
   {"2026-02-26", "utm_content", ""},
   {"2026-02-26", "utm_medium", "deeplink"},
   {"2026-02-26", "utm_medium", "partner"},
   {"2026-02-26", "utm_source", "justWatch-v2-catalog"},
   {"2026-02-26", "utm_source", "justwatch"},
   {"2026-02-26", "utm_source", "universal_search"},
   {"2026-02-26", "utm_term", ""},
}

func getUrlGroupingKey(rawUrl string) string {
   trimmedUrl := strings.TrimSuffix(rawUrl, "\n")
   parsed, err := url.Parse(trimmedUrl)
   if err != nil {
      return trimmedUrl
   }
   if parsed.RawQuery == "" {
      return parsed.String()
   }
   query := parsed.Query()
   for _, rule := range params_to_delete {
      // .Get() returns the first value. If the key doesn't exist, it returns "".
      // This perfectly handles the "assume one value" rule.
      if query.Get(rule.key) == rule.value {
         delete(query, rule.key)
      }
   }
   parsed.RawQuery = query.Encode()
   return parsed.String()
}
