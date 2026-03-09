// `presentationType` data seems to be incorrect in some cases. For example,
// JustWatch reports this as SD: fetchtv.com.au/movie/details/19285
// when the site itself reports as HD
package justWatch

import (
   "bytes"
   "cmp"
   "encoding/base64"
   "encoding/json"
   "errors"
   "maps"
   "net/http"
   "net/url"
   "slices"
   "strings"
   _ "embed"
)

//go:embed GetUrlTitleDetails.gql
var get_url_title_details string

//go:embed BackendConstantsFetcherQuery.gql
var backend_constants_fetcher_query string

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
func GroupAndSortByUrl(offers []*EnrichedOffer) ([]string, map[string][]*EnrichedOffer) {
   groupedOffers := make(map[string][]*EnrichedOffer)
   for _, offer := range offers {
      key := getUrlGroupingKey(offer.Offer.StandardWebUrl)
      groupedOffers[key] = append(groupedOffers[key], offer)
   }
   for _, offerGroup := range groupedOffers {
      slices.SortFunc(offerGroup, func(a, b *EnrichedOffer) int {
         return cmp.Compare(a.Locale.Country, b.Locale.Country)
      })
   }
   // This works for Go 1.21 and older.
   keys := slices.SortedFunc(maps.Keys(groupedOffers), func(a, b string) int {
      return cmp.Compare(len(a), len(b))
   })
   return keys, groupedOffers
}

// FilterOffers removes offers with unwanted monetization types.
func FilterOffers(offers []*EnrichedOffer, unwantedTypes ...string) []*EnrichedOffer {
   unwantedSet := make(map[string]struct{}, len(unwantedTypes))
   for _, unwanted := range unwantedTypes {
      unwantedSet[unwanted] = struct{}{}
   }
   var filteredOffers []*EnrichedOffer
   for _, offer := range offers {
      if _, found := unwantedSet[offer.Offer.MonetizationType]; !found {
         filteredOffers = append(filteredOffers, offer)
      }
   }
   return filteredOffers
}

type Offer struct {
   ElementCount     int
   MonetizationType string
   StandardWebUrl   string
}

type Locale struct {
   FullLocale  string
   Country     string
   CountryName string
}

type EnrichedOffer struct {
   Locale *Locale
   Offer  *Offer
}

// Deduplicate removes true duplicates where both the Offer and Locale are identical.
func Deduplicate(offers []*EnrichedOffer) []*EnrichedOffer {
   // 1. Sort the slice. This brings identical EnrichedOffers next to each other.
   // This part is correct as it compares the underlying values.
   slices.SortFunc(offers, func(a, b *EnrichedOffer) int {
      return cmp.Or(
         cmp.Compare(a.Offer.StandardWebUrl, b.Offer.StandardWebUrl),
         cmp.Compare(a.Offer.MonetizationType, b.Offer.MonetizationType),
         a.Offer.ElementCount-b.Offer.ElementCount,
         cmp.Compare(a.Locale.FullLocale, b.Locale.FullLocale),
      )
   })
   // 2. Compact the sorted slice, removing consecutive duplicates.
   return slices.CompactFunc(offers, func(a, b *EnrichedOffer) bool {
      return a.Offer.StandardWebUrl == b.Offer.StandardWebUrl &&
         a.Offer.MonetizationType == b.Offer.MonetizationType &&
         a.Offer.ElementCount == b.Offer.ElementCount &&
         a.Locale.FullLocale == b.Locale.FullLocale
   })
}

func (c *Content) Fetch(path string) error {
   var req http.Request
   req.Header = http.Header{}
   req.URL = &url.URL{
      Scheme:   "https",
      Host:     "apis.justwatch.com",
      Path:     "/content/urls",
      RawQuery: url.Values{"path": {path}}.Encode(),
   }
   resp, err := http.DefaultClient.Do(&req)
   if err != nil {
      return err
   }
   defer resp.Body.Close()
   if resp.StatusCode != http.StatusOK {
      return errors.New(resp.Status)
   }
   return json.NewDecoder(resp.Body).Decode(c)
}

type HrefLangTag struct {
   Href   string // /ar/pelicula/mulholland-drive
   Locale string // es_AR
}

// 2025-11-04
var EnUs = Locales{
   {FullLocale: "en_US", Country: "US", CountryName: "United States"},
   {FullLocale: "de_DE", Country: "DE", CountryName: "Germany"},
   {FullLocale: "pt_BR", Country: "BR", CountryName: "Brazil"},
   {FullLocale: "en_AU", Country: "AU", CountryName: "Australia"},
   {FullLocale: "en_NZ", Country: "NZ", CountryName: "New Zealand"},
   {FullLocale: "en_CA", Country: "CA", CountryName: "Canada"},
   {FullLocale: "en_GB", Country: "GB", CountryName: "United Kingdom"},
   {FullLocale: "en_ZA", Country: "ZA", CountryName: "South Africa"},
   {FullLocale: "en_IE", Country: "IE", CountryName: "Ireland"},
   {FullLocale: "en_BS", Country: "BS", CountryName: "Bahamas"},
   {FullLocale: "fr_GF", Country: "GF", CountryName: "French Guiana"},
   {FullLocale: "bs_BA", Country: "BA", CountryName: "Bosnia and Herzegovina"},
   {FullLocale: "it_VA", Country: "VA", CountryName: "Vatican City"},
   {FullLocale: "sq_XK", Country: "XK", CountryName: "Kosovo"},
   {FullLocale: "be_BY", Country: "BY", CountryName: "Belarus"},
   {FullLocale: "en_DK", Country: "DK", CountryName: "Denmark"},
   {FullLocale: "en_BZ", Country: "BZ", CountryName: "Belize"},
   {FullLocale: "el_CY", Country: "CY", CountryName: "Cyprus"},
   {FullLocale: "en_CM", Country: "CM", CountryName: "Cameroon"},
   {FullLocale: "en_GY", Country: "GY", CountryName: "Guyana"},
   {FullLocale: "fr_ML", Country: "ML", CountryName: "Mali"},
   {FullLocale: "es_NI", Country: "NI", CountryName: "Nicaragua"},
   {FullLocale: "fr_CD", Country: "CD", CountryName: "DR Congo"},
   {FullLocale: "en_MW", Country: "MW", CountryName: "Malawi"},
   {FullLocale: "sw_TZ", Country: "TZ", CountryName: "Tanzania"},
   {FullLocale: "en_PG", Country: "PG", CountryName: "Papua New Guinea"},
   {FullLocale: "en_ZW", Country: "ZW", CountryName: "Zimbabwe"},
   {FullLocale: "az_AZ", Country: "AZ", CountryName: "Azerbaijan"},
   {FullLocale: "lv_LV", Country: "LV", CountryName: "Latvia"},
   {FullLocale: "es_EC", Country: "EC", CountryName: "Ecuador"},
   {FullLocale: "zh_TW", Country: "TW", CountryName: "Taiwan"},
   {FullLocale: "ur_PK", Country: "PK", CountryName: "Pakistan"},
   {FullLocale: "bg_BG", Country: "BG", CountryName: "Bulgaria"},
   {FullLocale: "ru_RU", Country: "RU", CountryName: "Russia"},
   {FullLocale: "de_CH", Country: "CH", CountryName: "Switzerland"},
   {FullLocale: "de_AT", Country: "AT", CountryName: "Austria"},
   {FullLocale: "en_MY", Country: "MY", CountryName: "Malaysia"},
   {FullLocale: "en_SG", Country: "SG", CountryName: "Singapore"},
   {FullLocale: "fi_FI", Country: "FI", CountryName: "Finland"},
   {FullLocale: "hu_HU", Country: "HU", CountryName: "Hungary"},
   {FullLocale: "el_GR", Country: "GR", CountryName: "Greece"},
   {FullLocale: "es_CO", Country: "CO", CountryName: "Colombia"},
   {FullLocale: "uk_UA", Country: "UA", CountryName: "Ukraine"},
   {FullLocale: "es_HN", Country: "HN", CountryName: "Honduras"},
   {FullLocale: "et_EE", Country: "EE", CountryName: "Estonia"},
   {FullLocale: "es_PY", Country: "PY", CountryName: "Paraguay"},
   {FullLocale: "is_IS", Country: "IS", CountryName: "Iceland"},
   {FullLocale: "es_PA", Country: "PA", CountryName: "Panama"},
   {FullLocale: "es_UY", Country: "UY", CountryName: "Uruguay"},
   {FullLocale: "es_DO", Country: "DO", CountryName: "Dominican Republic"},
   {FullLocale: "es_ES", Country: "ES", CountryName: "Spain"},
   {FullLocale: "fr_FR", Country: "FR", CountryName: "France"},
   {FullLocale: "ar_EG", Country: "EG", CountryName: "Egypt"},
   {FullLocale: "ar_AE", Country: "AE", CountryName: "United Arab Emirates"},
   {FullLocale: "ar_IQ", Country: "IQ", CountryName: "Iraq"},
   {FullLocale: "hr_HR", Country: "HR", CountryName: "Croatia"},
   {FullLocale: "fr_CI", Country: "CI", CountryName: "Ivory Coast"},
   {FullLocale: "pt_CV", Country: "CV", CountryName: "Cape Verde"},
   {FullLocale: "fr_PF", Country: "PF", CountryName: "French Polynesia"},
   {FullLocale: "en_LC", Country: "LC", CountryName: "Saint Lucia"},
   {FullLocale: "fr_LU", Country: "LU", CountryName: "Luxembourg"},
   {FullLocale: "fr_SC", Country: "SC", CountryName: "Seychelles"},
   {FullLocale: "fr_NE", Country: "NE", CountryName: "Niger"},
   {FullLocale: "sr_ME", Country: "ME", CountryName: "Montenegro"},
   {FullLocale: "fr_MG", Country: "MG", CountryName: "Madagascar"},
   {FullLocale: "pt_MZ", Country: "MZ", CountryName: "Mozambique"},
   {FullLocale: "en_KE", Country: "KE", CountryName: "Kenya"},
   {FullLocale: "en_UG", Country: "UG", CountryName: "Uganda"},
   {FullLocale: "en_TT", Country: "TT", CountryName: "Trinidad and Tobago"},
   {FullLocale: "en_TC", Country: "TC", CountryName: "Turks and Caicos Islands"},
   {FullLocale: "en_ZM", Country: "ZM", CountryName: "Zambia"},
   {FullLocale: "fr_SN", Country: "SN", CountryName: "Senegal"},
   {FullLocale: "en_JM", Country: "JM", CountryName: "Jamaica"},
   {FullLocale: "ar_LB", Country: "LB", CountryName: "Lebanon"},
   {FullLocale: "ar_PS", Country: "PS", CountryName: "Palestine"},
   {FullLocale: "mk_MK", Country: "MK", CountryName: "Macedonia"},
   {FullLocale: "es_CU", Country: "CU", CountryName: "Cuba"},
   {FullLocale: "pt_AO", Country: "AO", CountryName: "Angola"},
   {FullLocale: "en_AG", Country: "AG", CountryName: "Antigua and Barbuda"},
   {FullLocale: "es_SV", Country: "SV", CountryName: "El Salvador"},
   {FullLocale: "ar_DZ", Country: "DZ", CountryName: "Algeria"},
   {FullLocale: "ar_MA", Country: "MA", CountryName: "Morocco"},
   {FullLocale: "ca_AD", Country: "AD", CountryName: "Andorra"},
   {FullLocale: "sq_AL", Country: "AL", CountryName: "Albania"},
   {FullLocale: "ar_JO", Country: "JO", CountryName: "Jordan"},
   {FullLocale: "ar_BH", Country: "BH", CountryName: "Bahrain"},
   {FullLocale: "ar_KW", Country: "KW", CountryName: "Kuwait"},
   {FullLocale: "ar_OM", Country: "OM", CountryName: "Oman"},
   {FullLocale: "ar_QA", Country: "QA", CountryName: "Qatar"},
   {FullLocale: "fr_BE", Country: "BE", CountryName: "Belgium"},
   {FullLocale: "ja_JP", Country: "JP", CountryName: "Japan"},
   {FullLocale: "ko_KR", Country: "KR", CountryName: "South Korea"},
   {FullLocale: "ar_SA", Country: "SA", CountryName: "Saudi Arabia"},
   {FullLocale: "es_AR", Country: "AR", CountryName: "Argentina"},
   {FullLocale: "it_IT", Country: "IT", CountryName: "Italy"},
   {FullLocale: "en_NL", Country: "NL", CountryName: "Netherlands"},
   {FullLocale: "pt_PT", Country: "PT", CountryName: "Portugal"},
   {FullLocale: "tr_TR", Country: "TR", CountryName: "Turkey"},
   {FullLocale: "en_IN", Country: "IN", CountryName: "India"},
   {FullLocale: "es_MX", Country: "MX", CountryName: "Mexico"},
   {FullLocale: "fr_BF", Country: "BF", CountryName: "Burkina Faso"},
   {FullLocale: "es_CL", Country: "CL", CountryName: "Chile"},
   {FullLocale: "es_PE", Country: "PE", CountryName: "Peru"},
   {FullLocale: "en_TH", Country: "TH", CountryName: "Thailand"},
   {FullLocale: "sv_SE", Country: "SE", CountryName: "Sweden"},
   {FullLocale: "cs_CZ", Country: "CZ", CountryName: "Czech Republic"},
   {FullLocale: "en_ID", Country: "ID", CountryName: "Indonesia"},
   {FullLocale: "pl_PL", Country: "PL", CountryName: "Poland"},
   {FullLocale: "en_PH", Country: "PH", CountryName: "Philippines"},
   {FullLocale: "ro_RO", Country: "RO", CountryName: "Romania"},
   {FullLocale: "en_NO", Country: "NO", CountryName: "Norway"},
   {FullLocale: "es_BO", Country: "BO", CountryName: "Bolivia"},
   {FullLocale: "en_BB", Country: "BB", CountryName: "Barbados"},
   {FullLocale: "es_CR", Country: "CR", CountryName: "Costa Rica"},
   {FullLocale: "ar_TD", Country: "TD", CountryName: "Chad"},
   {FullLocale: "en_GH", Country: "GH", CountryName: "Ghana"},
   {FullLocale: "es_GQ", Country: "GQ", CountryName: "Equatorial Guinea"},
   {FullLocale: "en_FJ", Country: "FJ", CountryName: "Fiji"},
   {FullLocale: "en_GG", Country: "GG", CountryName: "Guernsey"},
   {FullLocale: "mt_MT", Country: "MT", CountryName: "Malta"},
   {FullLocale: "fr_MU", Country: "MU", CountryName: "Mauritius"},
   {FullLocale: "es_GT", Country: "GT", CountryName: "Guatemala"},
   {FullLocale: "lt_LT", Country: "LT", CountryName: "Lithuania"},
   {FullLocale: "sr_RS", Country: "RS", CountryName: "Serbia"},
   {FullLocale: "sl_SI", Country: "SI", CountryName: "Slovenia"},
   {FullLocale: "en_NG", Country: "NG", CountryName: "Nigeria"},
   {FullLocale: "sk_SK", Country: "SK", CountryName: "Slovakia"},
   {FullLocale: "he_IL", Country: "IL", CountryName: "Israel"},
   {FullLocale: "es_VE", Country: "VE", CountryName: "Venezuela"},
   {FullLocale: "ro_MD", Country: "MD", CountryName: "Moldova"},
   {FullLocale: "zh_HK", Country: "HK", CountryName: "Hong Kong"},
   {FullLocale: "de_LI", Country: "LI", CountryName: "Liechtenstein"},
   {FullLocale: "fr_MC", Country: "MC", CountryName: "Monaco"},
   {FullLocale: "it_SM", Country: "SM", CountryName: "San Marino"},
   {FullLocale: "en_GI", Country: "GI", CountryName: "Gibraltar"},
   {FullLocale: "ar_TN", Country: "TN", CountryName: "Tunisia"},
   {FullLocale: "ar_LY", Country: "LY", CountryName: "Libya"},
   {FullLocale: "en_BM", Country: "BM", CountryName: "Bermuda"},
   {FullLocale: "ar_YE", Country: "YE", CountryName: "Yemen"},
}

type Locales []Locale

func (l Locales) Locale(tag *HrefLangTag) (*Locale, bool) {
   for _, locale_var := range l {
      if locale_var.FullLocale == tag.Locale {
         return &locale_var, true
      }
   }
   return nil, false
}

func FetchLocales(language string) (Locales, error) {
   data, err := json.Marshal(map[string]any{
      "query": backend_constants_fetcher_query,
      "variables": map[string]string{
         "language": language,
      },
   })
   if err != nil {
      return nil, err
   }
   req, err := http.NewRequest(
      "POST", "https://apis.justwatch.com/graphql", bytes.NewReader(data),
   )
   if err != nil {
      return nil, err
   }
   req.Header.Set("content-type", "application/json")
   req.Header.Set(
      "device-id", base64.RawStdEncoding.EncodeToString(make([]byte, 16)),
   )
   resp, err := http.DefaultClient.Do(req)
   if err != nil {
      return nil, err
   }
   if resp.StatusCode != http.StatusOK {
      var data strings.Builder
      err = resp.Write(&data)
      if err != nil {
         return nil, err
      }
      return nil, errors.New(data.String())
   }
   defer resp.Body.Close()
   var result struct {
      Data struct {
         Locales Locales
      }
   }
   err = json.NewDecoder(resp.Body).Decode(&result)
   if err != nil {
      return nil, err
   }
   return result.Data.Locales, nil
}

func (h *HrefLangTag) Offers(localeVar *Locale) ([]Offer, error) {
   data, err := json.Marshal(map[string]any{
      "query": get_url_title_details,
      "variables": map[string]string{
         "country":  localeVar.Country,
         "fullPath": h.Href,
      },
   })
   if err != nil {
      return nil, err
   }
   resp, err := http.Post(
      "https://apis.justwatch.com/graphql", "application/json",
      bytes.NewReader(data),
   )
   if err != nil {
      return nil, err
   }
   if resp.StatusCode != http.StatusOK {
      var data strings.Builder
      err = resp.Write(&data)
      if err != nil {
         return nil, err
      }
      return nil, errors.New(data.String())
   }
   defer resp.Body.Close()
   var result struct {
      Data struct {
         Url struct {
            Node struct {
               Offers []Offer
            }
         }
      }
   }
   err = json.NewDecoder(resp.Body).Decode(&result)
   if err != nil {
      return nil, err
   }
   return result.Data.Url.Node.Offers, nil
}

// https://justwatch.com/us/movie/goodfellas
func GetPath(rawUrl string) (string, error) {
   u, err := url.Parse(rawUrl)
   if err != nil {
      return "", err
   }
   if u.Scheme == "" {
      return "", errors.New("invalid URL: scheme is missing")
   }
   return u.Path, nil
}

type Content struct {
   HrefLangTags []HrefLangTag `json:"href_lang_tags"`
}
