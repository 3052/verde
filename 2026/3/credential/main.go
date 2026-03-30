package main

import (
   "encoding/json"
   "flag"
   "fmt"
   "os"
)

func main() {
   // Define command-line flags (all single-byte names)
   host := flag.String("h", "", "Host to search for (e.g., amcplus.com)")
   key := flag.String("k", "", "Key to retrieve (e.g., password) - Optional")
   file := flag.String("f", "", "Save the JSON file location permanently")

   flag.Parse()

   // Execute core logic. If an error is returned, print to stderr and exit 1.
   if err := run(*host, *key, *file); err != nil {
      fmt.Fprintf(os.Stderr, "Error: %v\n", err)
      os.Exit(1)
   }
}

// run orchestrates the loading, validating, and searching of credentials
func run(host, key, file string) error {
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
   return searchAndPrint(credentials, host, key)
}
