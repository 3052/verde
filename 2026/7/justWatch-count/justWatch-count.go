package main

import (
   "bytes"
   _ "embed"
   "encoding/json"
   "flag"
   "fmt"
   "io"
   "log"
   "net/http"
   "os"
   "sort"
   "strings"
)

// Global constant for the minimum Rotten Tomatoes score
const TomatoMeterMin = 50

//go:embed GetUrlMetadata.graphql
var metadataQuery string

// Global cache to prevent re-downloading the heavy provider list for the same locale
var providerCache = make(map[string][]Provider)

//go:embed GetProviderTop10TitlesFallback.graphql
var titlesQuery string

// executeGraphQL helper function
func executeGraphQL(payload map[string]interface{}) ([]byte, error) {
   jsonData, err := json.Marshal(payload)
   if err != nil {
      return nil, err
   }

   req, err := http.NewRequest("POST", "https://apis.justwatch.com/graphql", bytes.NewBuffer(jsonData))
   if err != nil {
      return nil, err
   }

   req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:140.0) Gecko/20100101 Firefox/140.0")
   req.Header.Set("Accept", "application/graphql-response+json,application/json;q=0.9")
   req.Header.Set("Content-Type", "application/json")
   req.Header.Set("Origin", "https://www.justwatch.com")

   client := &http.Client{}
   resp, err := client.Do(req)
   if err != nil {
      return nil, err
   }
   defer resp.Body.Close()

   bodyBytes, err := io.ReadAll(resp.Body)
   if err != nil {
      return nil, err
   }

   if resp.StatusCode != http.StatusOK {
      return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(bodyBytes))
   }
   return bodyBytes, nil
}

// fetchLocaleFromPath runs GetUrlMetadata to find the JustWatch locale (e.g. "en_US")
func fetchLocaleFromPath(path string) (string, error) {
   payload := map[string]interface{}{
      "operationName": "GetUrlMetadata",
      "variables": map[string]interface{}{
         "fullPath": path,
         "site":     "www",
      },
      "query": metadataQuery,
   }

   body, err := executeGraphQL(payload)
   if err != nil {
      return "", err
   }

   var resp struct {
      Data struct {
         UrlV2 struct {
            Locale string `json:"locale"`
         } `json:"urlV2"`
      } `json:"data"`
   }

   if err := json.Unmarshal(body, &resp); err != nil {
      return "", err
   }
   if resp.Data.UrlV2.Locale == "" {
      return "", fmt.Errorf("API returned empty locale for path")
   }

   return resp.Data.UrlV2.Locale, nil
}

// fetchTotalCount runs the GetProviderTop10TitlesFallback query to get the catalog size
func fetchTotalCount(pkg, country string) (int, error) {
   variables := map[string]interface{}{
      "country":             country,
      "first":               0,
      "popularTitlesSortBy": "TRENDING",
      "popularTitlesFilter": map[string]interface{}{
         "packages": []string{pkg},
         "tomatoMeter": map[string]int{
            "min": TomatoMeterMin,
         },
      },
   }

   payload := map[string]interface{}{
      "operationName": "GetProviderTop10TitlesFallback",
      "variables":     variables,
      "query":         titlesQuery,
   }

   body, err := executeGraphQL(payload)
   if err != nil {
      return 0, err
   }

   var resp struct {
      Errors []struct {
         Message string `json:"message"`
      } `json:"errors"`
      Data *struct {
         PopularTitles struct {
            TotalCount int `json:"totalCount"`
         } `json:"popularTitles"`
      } `json:"data"`
   }

   if err := json.Unmarshal(body, &resp); err != nil {
      return 0, err
   }
   if len(resp.Errors) > 0 {
      return 0, fmt.Errorf("GraphQL error: %s", resp.Errors[0].Message)
   }
   if resp.Data == nil {
      return 0, fmt.Errorf("API returned null data")
   }

   return resp.Data.PopularTitles.TotalCount, nil
}

func main() {
   fileFlag := flag.String("file", "", "Path to JSON file containing an array of objects (required)")
   flag.Parse()

   if *fileFlag == "" {
      flag.Usage()
      log.Fatal("Error: the -file flag is required.")
   }

   fileBytes, err := os.ReadFile(*fileFlag)
   if err != nil {
      log.Fatalf("Failed to read file '%s': %v", *fileFlag, err)
   }

   var inputs []ProviderInput
   if err := json.Unmarshal(fileBytes, &inputs); err != nil {
      log.Fatalf("Failed to parse JSON data: %v", err)
   }

   var results []Result

   for i, input := range inputs {
      path := strings.TrimSpace(input.Path)
      if path == "" {
         continue
      }
      date := strings.TrimSpace(input.Date)

      // Progress indicator (writes to stderr)
      log.Printf("[%d/%d] Processing: %s...", i+1, len(inputs), path)

      // 1. Get Locale from Path
      locale, err := fetchLocaleFromPath(path)
      if err != nil {
         log.Fatalf("[%s] Failed to get metadata: %v\n", path, err)
      }

      parts := strings.Split(locale, "_")
      if len(parts) != 2 {
         log.Fatalf("[%s] Invalid locale format received: %s\n", path, locale)
      }
      // Extract country directly from the path to match user examples (e.g. /uk/ -> UK)
      pathSegments := strings.Split(strings.Trim(path, "/"), "/")
      displayCountry := strings.ToUpper(pathSegments[0])

      // 2. Resolve the Package Code & Clear Name using the Locale & Path
      pkgCode, clearName, err := resolvePackageFromPath(path, locale)
      if err != nil {
         log.Fatalf("[%s] Failed to resolve package code: %v\n", path, err)
      }

      // 3. Fetch the Total Count (using the API's actual country code from locale, e.g. US, GB, CZ)
      apiCountry := parts[1]
      totalCount, err := fetchTotalCount(pkgCode, apiCountry)
      if err != nil {
         log.Fatalf("[%s] Failed to fetch total count: %v\n", path, err)
      }

      // Show completion for this item (writes to stderr)
      log.Printf("[%d/%d] Done: [%s] %s -> %d titles", i+1, len(inputs), displayCountry, clearName, totalCount)

      results = append(results, Result{
         Count:     totalCount,
         Country:   displayCountry,
         ClearName: clearName,
         Path:      path,
         Date:      date,
      })
   }

   // --- First Table: Sorted by Title Count DESC ---
   sort.Slice(results, func(i, j int) bool {
      return results[i].Count > results[j].Count
   })

   fmt.Println("\n| Titles | Country | Provider |")
   fmt.Println("|---|---|---|")
   for _, r := range results {
      fmt.Printf("| %d | %s | [%s] |\n", r.Count, r.Country, r.ClearName)
   }

   // --- Second Table: Sorted by Date DESC with chronological Yearly Count ---

   // 1. Sort ASC by Date to assign chronological counts (oldest = 1)
   sort.Slice(results, func(i, j int) bool {
      return results[i].Date < results[j].Date
   })

   var currentYear string
   var yearCount int
   for i := range results {
      year := ""
      if len(results[i].Date) >= 4 {
         year = results[i].Date[:4]
      }

      if year != currentYear {
         currentYear = year
         yearCount = 1
      } else {
         yearCount++
      }
      results[i].YearRank = yearCount
   }

   // 2. Sort DESC by Date for display
   sort.Slice(results, func(i, j int) bool {
      return results[i].Date > results[j].Date
   })

   fmt.Println("\n| Date | Country | Provider | Count |")
   fmt.Println("|---|---|---|---|")

   for _, r := range results {
      // Printed without Markdown link brackets for the provider
      fmt.Printf("| %s | %s | %s | %d |\n", r.Date, r.Country, r.ClearName, r.YearRank)
   }

   fmt.Println()

   // Print Markdown Links (writes to stdout)
   // Used only by the first table
   for _, r := range results {
      fmt.Printf("[%s]:https://justwatch.com%s?tomatoMeter=%d\n", r.ClearName, r.Path, TomatoMeterMin)
   }
}

// resolvePackageFromPath fetches the provider list (using cache) and matches the slug
func resolvePackageFromPath(path, locale string) (string, string, error) {
   cleanPath := strings.TrimRight(path, "/")
   segments := strings.Split(cleanPath, "/")
   urlSlug := segments[len(segments)-1]

   providers, cached := providerCache[locale]

   if !cached {
      restURL := fmt.Sprintf("https://apis.justwatch.com/content/providers/locale/%s", locale)

      req, err := http.NewRequest("GET", restURL, nil)
      if err != nil {
         return "", "", err
      }
      req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:140.0) Gecko/20100101 Firefox/140.0")

      client := &http.Client{}
      resp, err := client.Do(req)
      if err != nil {
         return "", "", err
      }
      defer resp.Body.Close()

      if resp.StatusCode != 200 {
         return "", "", fmt.Errorf("REST API returned status %d", resp.StatusCode)
      }

      if err := json.NewDecoder(resp.Body).Decode(&providers); err != nil {
         return "", "", err
      }

      providerCache[locale] = providers
   }

   for _, p := range providers {
      if p.Slug == urlSlug {
         return p.ShortName, p.ClearName, nil
      }
   }

   return "", "", fmt.Errorf("could not find provider with slug '%s' in locale '%s'", urlSlug, locale)
}

// Provider models the JustWatch REST API response
type Provider struct {
   ShortName     string `json:"short_name"`
   TechnicalName string `json:"technical_name"`
   ClearName     string `json:"clear_name"`
   Slug          string `json:"slug"`
}

// Struct to read the updated JSON format
type ProviderInput struct {
   Path string `json:"path"`
   Date string `json:"date"`
}

// Result models the final data for our Markdown tables
type Result struct {
   Count     int
   Country   string
   ClearName string
   Path      string
   Date      string
   YearRank  int
}
