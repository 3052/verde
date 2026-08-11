package main

import (
   "encoding/json"
   "flag"
   "fmt"
   "log"
   "os"
   "path/filepath"

   "41.neocities.org/scarlet"
)

func main() {
   log.SetFlags(log.Ltime)

   apiKeyFlag := flag.String("api-key", "", "Update the API key in your config")
   apiUrlFlag := flag.String("api-url", "", "Update the API URL in your config (expects an OpenAI-compatible chat completions endpoint)")
   modelFlag := flag.String("model", "", "Update the Model name in your config")
   serveFlag := flag.Bool("serve", false, "Start the local chatbot server")

   flag.Parse()

   // Strictly enforce exactly one flag at a time
   if flag.NFlag() != 1 || flag.NArg() > 0 {
      fmt.Println("Error: You must provide exactly one valid flag at a time.")
      flag.Usage()
      os.Exit(1)
   }

   if err := run(apiKeyFlag, apiUrlFlag, modelFlag, *serveFlag); err != nil {
      log.Fatal(err)
   }
}

func run(apiKeyFlag, apiUrlFlag, modelFlag *string, serve bool) error {
   configDir, err := os.UserConfigDir()
   if err != nil {
      return fmt.Errorf("error getting user config directory: %w", err)
   }

   appConfigDir := filepath.Join(configDir, "scarlet")
   configFilePath := filepath.Join(appConfigDir, "config.json")

   // Start with an empty config
   cfg := &scarlet.AppConfig{}

   // Try to load existing config
   if data, err := os.ReadFile(configFilePath); err == nil {
      if err := json.Unmarshal(data, cfg); err != nil {
         log.Printf("warning: error parsing existing config.json: %v", err)
      }
   }

   // Update config ONLY if flags were explicitly provided by the user in the CLI
   updated := false
   flag.Visit(func(f *flag.Flag) {
      switch f.Name {
      case "api-key":
         cfg.APIKey = *apiKeyFlag
         updated = true
      case "api-url":
         cfg.APIURL = *apiUrlFlag
         updated = true
      case "model":
         cfg.Model = *modelFlag
         updated = true
      }
   })

   if updated {
      if err := os.MkdirAll(appConfigDir, 0700); err != nil {
         return fmt.Errorf("error creating config directory: %w", err)
      }

      configData, err := json.MarshalIndent(cfg, "", "  ")
      if err != nil {
         return fmt.Errorf("error marshaling config data: %w", err)
      }

      if err := os.WriteFile(configFilePath, configData, 0600); err != nil {
         return fmt.Errorf("error writing config file: %w", err)
      }

      log.Printf("Wrote %s", configFilePath)
   }

   // If the user did not explicitly request to start the server, exit here
   if !serve {
      return nil
   }

   // Ensure all required configuration fields are present before starting
   if cfg.APIKey == "" {
      return fmt.Errorf("api key not found; please run with '-api-key YOUR_KEY'")
   }
   if cfg.APIURL == "" {
      return fmt.Errorf("api url not found; please run with '-api-url YOUR_URL'")
   }
   if cfg.Model == "" {
      return fmt.Errorf("model not found; please run with '-model YOUR_MODEL'")
   }

   return scarlet.RunServer(cfg)
}
