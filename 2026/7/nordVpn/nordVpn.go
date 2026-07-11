package main

import (
   "bufio"
   "cmp"
   "encoding/json"
   "fmt"
   "io"
   "net/http"
   "net/url"
   "os"
   "os/exec"
   "path/filepath"
   "slices"
   "strings"
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

func loadUsedServers(path string) map[string]bool {
   used := make(map[string]bool)
   file, err := os.Open(path)
   if err != nil {
      return used
   }
   defer file.Close()
   scanner := bufio.NewScanner(file)
   for scanner.Scan() {
      line := strings.TrimSpace(scanner.Text())
      if line != "" {
         used[line] = true
      }
   }
   return used
}

func processCountryServers(filePath, cacheDir, country, downloadURL string) error {
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

   usedPath := usedServersPath(cacheDir)
   used := loadUsedServers(usedPath)

   var candidates []*Server
   for _, server := range filtered {
      if !used[server.Hostname] {
         candidates = append(candidates, server)
      }
   }

   if len(candidates) == 0 {
      return fmt.Errorf(
         "all %d servers for %s have been used. Run: %s -reset",
         len(filtered), country, os.Args[0],
      )
   }

   // Limit to first 10 candidates
   if len(candidates) > 10 {
      candidates = candidates[:10]
   }

   fmt.Fprintf(os.Stderr, "Testing %d candidate(s) for %s (%d used globally)…\n\n",
      len(candidates), country, len(used))

   username, password, err := getCredentials()
   if err != nil {
      return fmt.Errorf("failed to retrieve credentials: %w", err)
   }

   const (
      probeTimeout   = 30 * time.Second
      rateLimitDelay = 210 * time.Second
   )

   var tested []string

   type result struct {
      server *Server
      speed  float64
      url    string
   }
   var results []result

   for _, server := range candidates {
      proxyHostname := server.Hostname
      for _, technology := range server.Technologies {
         for _, meta := range technology.Metadata {
            if meta.Name == "proxy_hostname" && meta.Value != "" {
               proxyHostname = meta.Value
            }
         }
      }

      proxyURL := url.URL{
         Scheme: "https",
         User:   url.UserPassword(username, password),
         Host:   fmt.Sprintf("%s:89", proxyHostname),
      }

      speedBps, rateLimited, err := testProxy(proxyURL.String(), downloadURL, probeTimeout)

      if rateLimited {
         fmt.Fprintf(os.Stderr, "RATE  %-20s  rate-limited, waiting %s and retrying…\n", server.Name, rateLimitDelay)
         time.Sleep(rateLimitDelay)
         speedBps, rateLimited, err = testProxy(proxyURL.String(), downloadURL, probeTimeout)
      }

      if rateLimited {
         fmt.Fprintf(os.Stderr, "\nStill rate-limited after retry. Saving %d tested server(s)…\n", len(tested))
         for _, hostname := range tested {
            if saveErr := saveUsedServer(usedPath, hostname); saveErr != nil {
               fmt.Fprintf(os.Stderr, "Warning: could not save used-server state for %s: %v\n", hostname, saveErr)
            }
         }
         return fmt.Errorf("rate-limited by NordVPN. %d server(s) saved as tested — re-run to continue", len(tested))
      }

      tested = append(tested, server.Hostname)

      if err != nil {
         fmt.Fprintf(os.Stderr, "SKIP  %-20s  %v\n", server.Name, err)
         continue
      }

      speedMB := speedBps / 1024 / 1024
      fmt.Fprintf(os.Stderr, "OK    %-20s  %.1f MB/s\n", server.Name, speedMB)

      results = append(results, result{
         server: server,
         speed:  speedBps,
         url:    proxyURL.String(),
      })
   }

   for _, hostname := range tested {
      if saveErr := saveUsedServer(usedPath, hostname); saveErr != nil {
         fmt.Fprintf(os.Stderr, "Warning: could not save used-server state for %s: %v\n", hostname, saveErr)
      }
   }

   if len(results) == 0 {
      return fmt.Errorf("no candidate server responded successfully")
   }

   slices.SortFunc(results, func(a, b result) int {
      return cmp.Compare(a.speed, b.speed)
   })

   for i, r := range results {
      speedMB := r.speed / 1024 / 1024
      if i > 0 {
         fmt.Println()
      }
      fmt.Printf("name: %s\n", r.server.Name)
      fmt.Printf("hostname: %s\n", r.server.Hostname)
      fmt.Printf("speed_mbps: %.1f\n", speedMB)
      fmt.Printf("url: %s\n", r.url)
   }

   return nil
}

func saveUsedServer(path, hostname string) error {
   file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
   if err != nil {
      return err
   }
   defer file.Close()
   _, err = fmt.Fprintln(file, hostname)
   return err
}

func testProxy(proxyURL, downloadURL string, timeout time.Duration) (float64, bool, error) {
   parsed, err := url.Parse(proxyURL)
   if err != nil {
      return 0, false, fmt.Errorf("bad proxy URL: %w", err)
   }

   transport := &http.Transport{
      Proxy: http.ProxyURL(parsed),
   }
   client := &http.Client{
      Transport: transport,
      Timeout:   timeout,
   }

   start := time.Now()
   resp, err := client.Get(downloadURL)
   if err != nil {
      if strings.Contains(err.Error(), "Proxy Authentication Required") {
         return 0, true, nil
      }
      return 0, false, fmt.Errorf("download test failed: %w", err)
   }
   defer resp.Body.Close()

   if resp.StatusCode != http.StatusOK {
      return 0, false, fmt.Errorf("download returned %s", resp.Status)
   }

   written, err := io.Copy(io.Discard, resp.Body)
   elapsed := time.Since(start)

   if err != nil {
      return 0, false, fmt.Errorf("download interrupted: %w", err)
   }

   if elapsed > 0 {
      return float64(written) / elapsed.Seconds(), false, nil
   }

   return 0, false, nil
}

func usedServersPath(cacheDir string) string {
   return filepath.Join(cacheDir, "nordVpn", "used.txt")
}

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

func filterServers(servers []*Server, countryCode string) []*Server {
   var filtered []*Server

   badGroups := map[string]bool{
      "legacy_dedicated_ip":       true,
      "legacy_double_vpn":         true,
      "legacy_obfuscated_servers": true,
   }

   for _, server := range servers {
      isTargetCountry := false
      for _, location := range server.Locations {
         if location.Country.Code == countryCode {
            isTargetCountry = true
            break
         }
      }

      isBadServer := false
      for _, group := range server.Groups {
         if badGroups[group.Identifier] {
            isBadServer = true
            break
         }
      }

      if isTargetCountry && !isBadServer {
         filtered = append(filtered, server)
      }
   }

   slices.SortFunc(filtered, func(a, b *Server) int {
      return cmp.Compare(a.Hostname, b.Hostname)
   })

   return filtered
}

type Technology struct {
   Identifier string     `json:"identifier"`
   Metadata   []Metadata `json:"metadata"`
}
