package main

import (
   "encoding/json"
   "flag"
   "fmt"
   "os"
   "path/filepath"
)

// AppConfig stores the user's saved preferences
type AppConfig struct {
   DataFile string `json:"data_file"`
}

// getConfigPath determines where to save/load the configuration file
func getConfigPath() (string, error) {
   configDir, err := os.UserConfigDir()
   if err != nil {
      return "", err
   }
   // e.g., ~/.config/credential/config.json on Mac/Linux
   appConfigDir := filepath.Join(configDir, "credential")
   return filepath.Join(appConfigDir, "config.json"), nil
}

func main() {
   // Define command-line flags
   host := flag.String("h", "", "Host to search for (e.g., amcplus.com)")
   key := flag.String("k", "", "Key to retrieve (e.g., password)")
   setFile := flag.String("set-file", "", "Save the JSON file location permanently")

   flag.Parse()

   // 1. Get the path to the user's config file
   configPath, err := getConfigPath()
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error locating user config directory: %v\n", err)
      os.Exit(1)
   }

   // 2. Handle saving the file location if --set-file is provided
   if *setFile != "" {
      // Convert to absolute path so it works from any directory
      absPath, err := filepath.Abs(*setFile)
      if err != nil {
         fmt.Fprintf(os.Stderr, "Error getting absolute path: %v\n", err)
         os.Exit(1)
      }

      cfg := AppConfig{DataFile: absPath}
      data, _ := json.MarshalIndent(cfg, "", "  ")

      // Create the config directory if it doesn't exist
      os.MkdirAll(filepath.Dir(configPath), 0755)

      // Write the config file
      if err := os.WriteFile(configPath, data, 0644); err != nil {
         fmt.Fprintf(os.Stderr, "Failed to save config: %v\n", err)
         os.Exit(1)
      }

      fmt.Printf("Successfully saved data file location: %s\n", absPath)
      os.Exit(0)
   }

   // 3. Load the data file location from the config
   var dataFile string
   if b, err := os.ReadFile(configPath); err == nil {
      var cfg AppConfig
      if err := json.Unmarshal(b, &cfg); err == nil {
         dataFile = cfg.DataFile
      }
   }

   // If the config file doesn't exist or is empty, prompt the user
   if dataFile == "" {
      fmt.Fprintln(os.Stderr, "Error: No data file location configured.")
      fmt.Fprintln(os.Stderr, "Please use '--set-file <path>' to save your JSON location first.")
      os.Exit(1)
   }

   // 4. Validate host and key flags
   if *host == "" || *key == "" {
      fmt.Fprintln(os.Stderr, "Usage: credential -h <host> -k <key>")
      os.Exit(1)
   }

   // 5. Read the target credentials JSON file
   data, err := os.ReadFile(dataFile)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error reading credentials file '%s': %v\n", dataFile, err)
      os.Exit(1)
   }

   // 6. Parse and Search
   var credentials []map[string]interface{}
   if err := json.Unmarshal(data, &credentials); err != nil {
      fmt.Fprintf(os.Stderr, "Error parsing JSON: %v\n", err)
      os.Exit(1)
   }

   for _, cred := range credentials {
      if credHost, ok := cred["host"].(string); ok && credHost == *host {
         if val, exists := cred[*key]; exists {
            fmt.Printf("%v\n", val)
            os.Exit(0)
         }
      }
   }

   fmt.Fprintf(os.Stderr, "Could not find key '%s' for host '%s'\n", *key, *host)
   os.Exit(1)
}
