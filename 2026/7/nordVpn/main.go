package main

import (
   "flag"
   "fmt"
   "io"
   "log"
   "net/http"
   "net/url"
   "os"
   "path/filepath"
)

func main() {
   log.SetFlags(log.Ltime)
   refresh := flag.Bool("refresh", false, "Fetch the latest server list from NordVPN")
   country := flag.String("country", "", "Target country code (e.g., PL, DE, US)")
   reset := flag.Bool("reset", false, "Reset the used-servers list (all countries)")
   downloadURL := flag.String("download", "", "URL to test download speed with (required for -country)")
   flag.Parse()

   cacheDir, err := os.UserCacheDir()
   if err != nil {
      log.Fatalf("Failed to get user cache directory: %v", err)
   }
   filePath := filepath.Join(cacheDir, "nordVpn", "nordVpn.json")

   if *refresh {
      if err := refreshFile(filePath); err != nil {
         log.Fatalf("Refresh failed: %v", err)
      }
      return
   }

   if *reset {
      usedPath := usedServersPath(cacheDir)
      if err := os.Remove(usedPath); err != nil && !os.IsNotExist(err) {
         log.Fatalf("Failed to reset used servers: %v", err)
      }
      fmt.Fprintf(os.Stderr, "Used-servers list reset (all countries).\n")
      return
   }

   if *country != "" {
      if *downloadURL == "" {
         log.Fatalf("-country requires -download URL")
      }
      if err := processCountryServers(filePath, cacheDir, *country, *downloadURL); err != nil {
         log.Fatalf("Error: %v", err)
      }
      return
   }

   fmt.Fprintf(os.Stderr, "Error: You must provide either -refresh, -reset, or -country.\n\n")
   flag.Usage()
   os.Exit(1)
}

func refreshFile(filePath string) error {
   endpoint := url.URL{
      Scheme:   "https",
      Host:     "api.nordvpn.com",
      Path:     "/v1/servers",
      RawQuery: "limit=0",
   }

   fmt.Fprintf(os.Stderr, "Downloading latest server list from %s...\n", endpoint.String())
   resp, err := http.Get(endpoint.String())
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
