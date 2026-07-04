package main

import (
   "cmp"
   "encoding/json"
   "flag"
   "fmt"
   "io"
   "log"
   "net/http"
   "net/url"
   "os"
   "os/exec"
   "path/filepath"
   "slices"
   "time"
)

func getCredentials() (string, string, error) {
   cmd := exec.Command("credential", "-j=api.nordvpn.com")
   output, err := cmd.Output()
   if err != nil {
      return "", "", fmt.Errorf("failed to run credential command: %w", err)
   }

   var creds []struct {
      Username string
      Password string
   }

   if err := json.Unmarshal(output, &creds); err != nil {
      return "", "", fmt.Errorf("failed to parse credentials JSON: %w", err)
   }

   if len(creds) == 0 {
      return "", "", fmt.Errorf("no credentials found in command output")
   }

   return creds[0].Username, creds[0].Password, nil
}

func main() {
   log.SetFlags(log.Ltime)
   refresh := flag.Bool("refresh", false, "Fetch the latest server list from NordVPN")
   country := flag.String("country", "", "Target country code (e.g., PL, DE, US)")
   flag.Parse()

   cacheDir, err := os.UserCacheDir()
   if err != nil {
      log.Fatalf("Failed to get user cache directory: %v", err)
   }
   filePath := filepath.Join(cacheDir, "nordVpn", "nordVpn.json")

   // ACTION 1: Refresh
   if *refresh {
      if err := refreshFile(filePath); err != nil {
         log.Fatalf("Refresh failed: %v", err)
      }
      return
   }

   // ACTION 2: Get Country Servers
   if *country != "" {
      if err := processCountryServers(filePath, *country); err != nil {
         log.Fatalf("Error processing country servers: %v", err)
      }
      return
   }

   // If neither flag was provided
   fmt.Fprintf(os.Stderr, "Error: You must provide either -refresh or -country.\n\n")
   flag.Usage()
   os.Exit(1)
}

func processCountryServers(filePath string, country string) error {
   fileInfo, err := os.Stat(filePath)
   if err != nil {
      if os.IsNotExist(err) {
         return fmt.Errorf("cache file does not exist. Please run the program with the -refresh flag first")
      }
      return fmt.Errorf("failed to access file %s: %w", filePath, err)
   }

   // If the file was modified 24 hours ago or more, prompt user to refresh
   if time.Since(fileInfo.ModTime()) >= 24*time.Hour {
      return fmt.Errorf("the file %s is 24 hours old or more. Please run with the -refresh flag", filePath)
   }

   jsonData, err := os.ReadFile(filePath)
   if err != nil {
      return fmt.Errorf("failed to read file %s: %w", filePath, err)
   }

   var servers []*Server
   if err := json.Unmarshal(jsonData, &servers); err != nil {
      return fmt.Errorf("failed to parse JSON: %w", err)
   }

   sortedServers := GetSortedServers(servers, country)

   if len(sortedServers) == 0 {
      return nil
   }

   username, password, err := getCredentials()
   if err != nil {
      return fmt.Errorf("failed to retrieve credentials: %w", err)
   }

   // Print the results in strict key-value line output format to Stdout
   for i, s := range sortedServers {
      // Find the actual proxy hostname for this server if it exists
      proxyHostname := s.Hostname // Default fallback
      for _, tech := range s.Technologies {
         for _, m := range tech.Metadata {
            if m.Name == "proxy_hostname" && m.Value != "" {
               proxyHostname = m.Value
               break
            }
         }
      }

      // Using url.URL safely URL-encodes the password and formats the string
      u := url.URL{
         Scheme: "https",
         User:   url.UserPassword(username, password),
         Host:   fmt.Sprintf("%s:89", proxyHostname),
      }

      fmt.Printf("name: %s\n", s.Name)
      fmt.Printf("load: %d\n", s.Load)
      fmt.Printf("url: %s\n", u.String())

      // Print a blank line between items, except after the very last one
      if i < len(sortedServers)-1 {
         fmt.Println()
      }
   }

   return nil
}

func refreshFile(filePath string) error {
   u := url.URL{
      Scheme:   "https",
      Host:     "api.nordvpn.com",
      Path:     "/v1/servers",
      RawQuery: "limit=0",
   }

   // Print info to Stderr so Stdout remains clean for scripting
   fmt.Fprintf(os.Stderr, "Downloading latest server list from %s...\n", u.String())
   resp, err := http.Get(u.String())
   if err != nil {
      return fmt.Errorf("failed to fetch data: %w", err)
   }
   defer resp.Body.Close()

   if resp.StatusCode != http.StatusOK {
      return fmt.Errorf("unexpected HTTP status: %s", resp.Status)
   }

   // Ensure the cache directory exists before writing
   if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
      return fmt.Errorf("failed to create directory: %w", err)
   }

   out, err := os.Create(filePath)
   if err != nil {
      return fmt.Errorf("failed to create file: %w", err)
   }
   defer out.Close()

   if _, err := io.Copy(out, resp.Body); err != nil {
      return fmt.Errorf("failed to write data to file: %w", err)
   }

   fmt.Fprintln(os.Stderr, "Server list successfully updated.")
   return nil
}

// Define the necessary structs to parse the JSON data
type Country struct {
   Code string `json:"code"`
}

type Group struct {
   Identifier string `json:"identifier"`
}

type Location struct {
   Country Country `json:"country"`
}

type Metadata struct {
   Name  string `json:"name"`
   Value string `json:"value"`
}

type Server struct {
   Name         string       `json:"name"`
   Hostname     string       `json:"hostname"`
   Load         int          `json:"load"`
   Locations    []Location   `json:"locations"`
   Groups       []Group      `json:"groups"`
   Technologies []Technology `json:"technologies"`
}

// GetSortedServers filters by exact country code, excludes known bad groups, and sorts by load descending.
func GetSortedServers(servers []*Server, countryCode string) []*Server {
   var filtered []*Server

   // Define server groups that do not work with standard proxy auth on port 89
   badGroups := map[string]bool{
      "legacy_dedicated_ip":       true, // Dedicated IPs
      "legacy_double_vpn":         true, // Double VPN
      "legacy_obfuscated_servers": true, // Obfuscated servers
   }

   for _, s := range servers {
      // 1. EXACT match for country code
      isTargetCountry := false
      for _, loc := range s.Locations {
         if loc.Country.Code == countryCode {
            isTargetCountry = true
            break
         }
      }

      // 2. Exclude known bad options
      isBadServer := false
      for _, group := range s.Groups {
         if badGroups[group.Identifier] {
            isBadServer = true
            break
         }
      }

      // Only append if it's the target country AND not a known bad server type
      if isTargetCountry && !isBadServer {
         filtered = append(filtered, s)
      }
   }

   // 3. Sort the filtered pointers by Load (Descending)
   slices.SortFunc(filtered, func(a, b *Server) int {
      return cmp.Compare(b.Load, a.Load)
   })

   return filtered
}

type Technology struct {
   Identifier string     `json:"identifier"`
   Metadata   []Metadata `json:"metadata"`
}
