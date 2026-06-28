package main

import (
   "encoding/json"
   "flag"
   "fmt"
   "os"
   "path/filepath"
   "sort"
   "strings"
   "time"
)

// getConfigPath determines where to save/load the configuration file
func getConfigPath() (string, error) {
   configDir, err := os.UserConfigDir()
   if err != nil {
      return "", err
   }
   return filepath.Join(configDir, "journal/credential.json"), nil
}

// loadConfig reads the config file and returns the saved DataFile path
func loadConfig(configPath string) (string, error) {
   b, err := os.ReadFile(configPath)
   if err != nil {
      if os.IsNotExist(err) {
         return "", nil // Normal if the user hasn't run -f yet
      }
      return "", fmt.Errorf("reading config file: %w", err)
   }

   var cfg AppConfig
   if err := json.Unmarshal(b, &cfg); err != nil {
      return "", fmt.Errorf("parsing config file (it might be corrupted): %w", err)
   }

   return cfg.DataFile, nil
}

func main() {
   // Define command-line flags (all single-byte names)
   host := flag.String("h", "", "Host to search for (e.g., amcplus.com)")
   hostJSON := flag.String("j", "", "Host to search for and output in JSON format (e.g., amcplus.com)")
   file := flag.String("f", "", "Save the JSON file location permanently")

   flag.Parse()

   // Determine the target host and output format
   targetHost := *host
   jsonOut := false

   // If the user provided the JSON string flag, override the standard targetHost
   if *hostJSON != "" {
      targetHost = *hostJSON
      jsonOut = true
   }

   // Execute core logic. If an error is returned, print to stderr and exit 1.
   if err := run(targetHost, *file, jsonOut); err != nil {
      fmt.Fprintf(os.Stderr, "Error: %v\n", err)
      os.Exit(1)
   }
}

// run orchestrates the loading, validating, and searching of credentials
func run(host, file string, jsonOut bool) error {
   configPath, err := getConfigPath()
   if err != nil {
      return fmt.Errorf("locating user config directory: %w", err)
   }

   // 1. Handle saving the file location if -f is provided
   if file != "" {
      return saveConfig(file, configPath)
   }

   // 2. Load the data file location from the config
   dataFile, err := loadConfig(configPath)
   if err != nil {
      return err
   }
   if dataFile == "" {
      return fmt.Errorf("no data file location configured. Please use '-f <path>' first")
   }

   // 3. Validate host flag
   if host == "" {
      flag.Usage()
      return nil
   }

   // 4. Read the target credentials JSON file
   data, err := os.ReadFile(dataFile)
   if err != nil {
      return fmt.Errorf("reading credentials file '%s': %w", dataFile, err)
   }

   // 5. Parse the JSON (strictly as strings)
   var credentials []map[string]string
   if err := json.Unmarshal(data, &credentials); err != nil {
      return fmt.Errorf("parsing JSON data: %w", err)
   }

   // 6. Validate the JSON data rules
   if err := validateData(credentials); err != nil {
      return err
   }

   // 7. Search and output
   return searchAndPrint(credentials, host, jsonOut)
}

// saveConfig saves the file path to the user's config directory exactly as provided
func saveConfig(file, configPath string) error {
   cfg := AppConfig{DataFile: file}
   configData, err := json.MarshalIndent(cfg, "", "  ")
   if err != nil {
      return fmt.Errorf("encoding config data: %w", err)
   }

   if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
      return fmt.Errorf("creating config directory: %w", err)
   }

   if err := os.WriteFile(configPath, configData, 0644); err != nil {
      return fmt.Errorf("saving config file: %w", err)
   }

   // Print success to standard output (with a newline for readability)
   fmt.Printf("Successfully saved data file location: %s\n", file)
   return nil
}

// searchAndPrint finds the matching object(s) and prints them to standard
// output
func searchAndPrint(credentials []map[string]string, host string, jsonOut bool) error {
   // Collect all objects matching the host
   var matches []map[string]string
   for _, cred := range credentials {
      if strings.EqualFold(cred["host"], host) {
         matches = append(matches, cred)
      }
   }

   if len(matches) == 0 {
      return fmt.Errorf("could not find any entries for host '%s'", host)
   }

   // Output as JSON if requested
   if jsonOut {
      jsonData, err := json.MarshalIndent(matches, "", "  ")
      if err != nil {
         return fmt.Errorf("failed to encode JSON: %w", err)
      }
      fmt.Println(string(jsonData))
      return nil
   }

   // Format and print the matching objects in default text format
   for i, match := range matches {
      if i > 0 {
         // Add an empty line between multiple results for readability
         fmt.Println()
      }

      // Extract and sort the keys alphabetically for consistent output
      var keys []string
      for k := range match {
         keys = append(keys, k)
      }
      sort.Strings(keys)

      // Print out the key-value pairs
      for _, k := range keys {
         fmt.Printf("%s: %s\n", k, match[k])
      }
   }

   return nil
}

// validateData enforces the specific data rules on the JSON credentials
func validateData(credentials []map[string]string) error {
   passCounts := make(map[string]int)

   // First pass: Count password occurrences
   for _, cred := range credentials {
      if passVal, exists := cred["password"]; exists {
         passCounts[passVal]++
      }
   }

   // Calculate the date exactly 1 year ago from today
   oneYearAgo := time.Now().AddDate(-1, 0, 0)

   // Second pass: Validate each rule
   for i, cred := range credentials {
      hostStr, hostExists := cred["host"]
      if !hostExists {
         hostStr = "unknown/missing host"
      }

      // --- Rule 1: All objects must have a date, and it cannot be older than 1 year ---
      dateStr, dateExists := cred["date"]
      if !dateExists {
         return fmt.Errorf("validation error: object at index %d (host: %s) is missing 'date' key", i, hostStr)
      }

      // Parse date assuming YYYY-MM-DD format
      parsedDate, err := time.Parse("2006-01-02", dateStr)
      if err != nil {
         return fmt.Errorf("validation error: object at index %d (host: %s) has invalid date '%s' (expected YYYY-MM-DD)", i, hostStr, dateStr)
      }
      if parsedDate.Before(oneYearAgo) {
         return fmt.Errorf("validation error: object at index %d (host: %s) has a date '%s' older than 1 year", i, hostStr, dateStr)
      }

      // --- Rule 2: Passwords must be unique UNLESS unique="false" ---
      passVal, passExists := cred["password"]
      if passExists {
         uniqueVal, uniqueExists := cred["unique"]

         // Secure by Default: We assume uniqueness is REQUIRED.
         // It is only disabled if the key explicitly exists and equals "false".
         requireUnique := true
         if uniqueExists && strings.ToLower(uniqueVal) == "false" {
            requireUnique = false
         }

         if requireUnique {
            if passCounts[passVal] > 1 {
               return fmt.Errorf("validation error: object at index %d (host: %s) shares its password with another object, but requires a unique password", i, hostStr)
            }
         }
      }
   }

   return nil
}

// AppConfig stores the user's saved preferences
type AppConfig struct {
   DataFile string `json:"data_file"`
}
