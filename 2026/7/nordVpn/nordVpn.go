package main

import (
   "bufio"
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
   "strings"
   "time"
)

// ---------------------------------------------------------------------------
// Credentials
// ---------------------------------------------------------------------------

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

func loadUsedServers(path string) map[string]bool {
   used := make(map[string]bool)
   f, err := os.Open(path)
   if err != nil {
      return used
   }
   defer f.Close()
   scanner := bufio.NewScanner(f)
   for scanner.Scan() {
      line := strings.TrimSpace(scanner.Text())
      if line != "" {
         used[line] = true
      }
   }
   return used
}

// ---------------------------------------------------------------------------
// main
// ---------------------------------------------------------------------------

func main() {
   log.SetFlags(log.Ltime)
   refresh := flag.Bool("refresh", false, "Fetch the latest server list from NordVPN")
   country := flag.String("country", "", "Target country code (e.g., PL, DE, US)")
   reset := flag.Bool("reset", false, "Reset the used-servers list for the given -country")
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

   // ACTION 2: Reset used-servers list
   if *reset {
      if *country == "" {
         log.Fatalf("-reset requires -country")
      }
      usedPath := usedServersPath(cacheDir, *country)
      if err := os.Remove(usedPath); err != nil && !os.IsNotExist(err) {
         log.Fatalf("Failed to reset used servers: %v", err)
      }
      fmt.Fprintf(os.Stderr, "Reset used-servers list for %s.\n", *country)
      return
   }

   // ACTION 3: Get a country server
   if *country != "" {
      if err := processCountryServers(filePath, cacheDir, *country); err != nil {
         log.Fatalf("Error: %v", err)
      }
      return
   }

   // No flags provided
   fmt.Fprintf(os.Stderr, "Error: You must provide either -refresh or -country.\n\n")
   flag.Usage()
   os.Exit(1)
}

// ---------------------------------------------------------------------------
// Main processing
// ---------------------------------------------------------------------------

func processCountryServers(filePath, cacheDir, country string) error {
   fileInfo, err := os.Stat(filePath)
   if err != nil {
      if os.IsNotExist(err) {
         return fmt.Errorf("cache file does not exist. Run with -refresh first")
      }
      return fmt.Errorf("failed to access file %s: %w", filePath, err)
   }

   if time.Since(fileInfo.ModTime()) >= 24*time.Hour {
      return fmt.Errorf("cache file is 24h+ old. Run with -refresh first")
   }

   jsonData, err := os.ReadFile(filePath)
   if err != nil {
      return fmt.Errorf("failed to read file %s: %w", filePath, err)
   }

   var servers []*Server
   if err := json.Unmarshal(jsonData, &servers); err != nil {
      return fmt.Errorf("failed to parse JSON: %w", err)
   }

   filtered := filterServers(servers, country)
   if len(filtered) == 0 {
      return fmt.Errorf("no servers found for country %s", country)
   }

   // Load the set of already-used server hostnames
   usedPath := usedServersPath(cacheDir, country)
   used := loadUsedServers(usedPath)

   // Build candidate list, excluding used servers
   var candidates []*Server
   for _, s := range filtered {
      if !used[s.Hostname] {
         candidates = append(candidates, s)
      }
   }

   if len(candidates) == 0 {
      return fmt.Errorf(
         "all %d servers for %s have been used. Run: %s -reset -country %s",
         len(filtered), country, os.Args[0], country,
      )
   }

   fmt.Fprintf(os.Stderr, "Testing %d candidate server(s) for %s (%d already used)…\n\n",
      len(candidates), country, len(used))

   username, password, err := getCredentials()
   if err != nil {
      return fmt.Errorf("failed to retrieve credentials: %w", err)
   }

   const (
      probeTarget  = "https://api.nordvpn.com/v1/helpers/ips/echo"
      probeTimeout = 8 * time.Second
      goodEnough   = 1500 * time.Millisecond
   )

   // Track the best (lowest-latency) server that responded, in case nothing
   // meets the goodEnough threshold.
   var best struct {
      server  *Server
      latency time.Duration
      url     string
   }

   for _, s := range candidates {
      // Resolve the proxy hostname from technologies metadata
      proxyHostname := s.Hostname
      for _, tech := range s.Technologies {
         for _, m := range tech.Metadata {
            if m.Name == "proxy_hostname" && m.Value != "" {
               proxyHostname = m.Value
            }
         }
      }

      proxyURL := url.URL{
         Scheme: "http",
         User:   url.UserPassword(username, password),
         Host:   fmt.Sprintf("%s:89", proxyHostname),
      }

      latency, err := testProxy(proxyURL.String(), probeTarget, probeTimeout)
      if err != nil {
         fmt.Fprintf(os.Stderr, "SKIP  %-40s  %v\n", s.Name, err)
         continue
      }

      fmt.Fprintf(os.Stderr, "OK    %-40s  %s\n", s.Name, latency)

      if best.server == nil || latency < best.latency {
         best.server = s
         best.latency = latency
         best.url = proxyURL.String()
      }

      // Early exit: this server is good enough
      if latency <= goodEnough {
         break
      }
   }

   if best.server == nil {
      return fmt.Errorf("no candidate server passed the proxy test")
   }

   // Mark this server as used so the next run picks a different one
   if err := saveUsedServer(usedPath, best.server.Hostname); err != nil {
      fmt.Fprintf(os.Stderr, "Warning: could not save used-server state: %v\n", err)
   }

   // Output the chosen server to stdout
   fmt.Printf("name: %s\n", best.server.Name)
   fmt.Printf("hostname: %s\n", best.server.Hostname)
   fmt.Printf("latency_ms: %d\n", best.latency.Milliseconds())
   fmt.Printf("url: %s\n", best.url)

   return nil
}

// ---------------------------------------------------------------------------
// Refresh
// ---------------------------------------------------------------------------

func refreshFile(filePath string) error {
   u := url.URL{
      Scheme:   "https",
      Host:     "api.nordvpn.com",
      Path:     "/v1/servers",
      RawQuery: "limit=0",
   }

   fmt.Fprintf(os.Stderr, "Downloading latest server list from %s...\n", u.String())
   resp, err := http.Get(u.String())
   if err != nil {
      return fmt.Errorf("failed to fetch data: %w", err)
   }
   defer resp.Body.Close()

   if resp.StatusCode != http.StatusOK {
      return fmt.Errorf("unexpected HTTP status: %s", resp.Status)
   }

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

func saveUsedServer(path, hostname string) error {
   f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
   if err != nil {
      return err
   }
   defer f.Close()
   _, err = fmt.Fprintln(f, hostname)
   return err
}

// ---------------------------------------------------------------------------
// Proxy testing
// ---------------------------------------------------------------------------

func testProxy(proxyURL, targetURL string, timeout time.Duration) (time.Duration, error) {
   u, err := url.Parse(proxyURL)
   if err != nil {
      return 0, fmt.Errorf("bad proxy URL: %w", err)
   }

   transport := &http.Transport{
      Proxy: http.ProxyURL(u),
   }
   client := &http.Client{
      Transport: transport,
      Timeout:   timeout,
   }

   start := time.Now()
   resp, err := client.Get(targetURL)
   if err != nil {
      return 0, err
   }
   defer resp.Body.Close()
   latency := time.Since(start)

   if resp.StatusCode != http.StatusOK {
      return 0, fmt.Errorf("upstream returned %s", resp.Status)
   }

   io.Copy(io.Discard, resp.Body)
   return latency, nil
}

// ---------------------------------------------------------------------------
// Used-server tracking (state file)
// ---------------------------------------------------------------------------

func usedServersPath(cacheDir, country string) string {
   return filepath.Join(cacheDir, "nordVpn", fmt.Sprintf("used_%s.txt", country))
}

// ---------------------------------------------------------------------------
// Structs
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// Server filtering (no sorting by load — we probe live instead)
// ---------------------------------------------------------------------------

func filterServers(servers []*Server, countryCode string) []*Server {
   var filtered []*Server

   badGroups := map[string]bool{
      "legacy_dedicated_ip":       true,
      "legacy_double_vpn":         true,
      "legacy_obfuscated_servers": true,
   }

   for _, s := range servers {
      isTargetCountry := false
      for _, loc := range s.Locations {
         if loc.Country.Code == countryCode {
            isTargetCountry = true
            break
         }
      }

      isBadServer := false
      for _, group := range s.Groups {
         if badGroups[group.Identifier] {
            isBadServer = true
            break
         }
      }

      if isTargetCountry && !isBadServer {
         filtered = append(filtered, s)
      }
   }

   // Sort by hostname for deterministic ordering across runs
   slices.SortFunc(filtered, func(a, b *Server) int {
      return cmp.Compare(a.Hostname, b.Hostname)
   })

   return filtered
}

type Technology struct {
   Identifier string     `json:"identifier"`
   Metadata   []Metadata `json:"metadata"`
}
