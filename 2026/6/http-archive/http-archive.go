package main

import (
   "encoding/json"
   "flag"
   "fmt"
   "os"
   "regexp"
   "time"
)

type HAR struct {
   Log struct {
      Version string            `json:"version"`
      Creator interface{}       `json:"creator"`
      Pages   interface{}       `json:"pages,omitempty"`
      Entries []json.RawMessage `json:"entries"`
   } `json:"log"`
}

type PartialEntry struct {
   Response struct {
      Content struct {
         Text string `json:"text"`
      } `json:"content"`
      Headers []NameValuePair `json:"headers"`
      Cookies []NameValuePair `json:"cookies"`
   } `json:"response"`
}

type NameValuePair struct {
   Name  string `json:"name"`
   Value string `json:"value"`
}

func main() {
   var inputFile, pattern string

   flag.StringVar(&inputFile, "in", "", "Path to the input .har file (required)")
   flag.StringVar(&pattern, "pattern", "", "Regex pattern to match against response values (required)")

   flag.Parse()

   if inputFile == "" || pattern == "" {
      fmt.Fprintln(os.Stderr, "Error: Missing required arguments (-in and -pattern).")
      flag.PrintDefaults()
      os.Exit(1)
   }

   re, err := regexp.Compile(pattern)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Fatal error: Invalid regex pattern: %v\n", err)
      os.Exit(1)
   }

   outputFile := fmt.Sprintf("%d.har", time.Now().Unix())

   if err := processHAR(inputFile, outputFile, re); err != nil {
      fmt.Fprintf(os.Stderr, "Fatal error: %v\n", err)
      os.Exit(1)
   }
}

func processHAR(inputFile, outputFile string, re *regexp.Regexp) error {
   data, err := os.ReadFile(inputFile)
   if err != nil {
      return fmt.Errorf("error reading file: %w", err)
   }

   var har HAR
   if err := json.Unmarshal(data, &har); err != nil {
      return fmt.Errorf("error parsing HAR JSON: %w", err)
   }

   var keptEntries []json.RawMessage

   // Iterate through all entries in the HAR
   for i, rawEntry := range har.Log.Entries {
      var entry PartialEntry
      if err := json.Unmarshal(rawEntry, &entry); err != nil {
         fmt.Fprintf(os.Stderr, "Warning: failed to parse entry %d: %v\n", i, err)
         continue
      }

      matched := false

      // 1. Check Response Cookies
      for _, c := range entry.Response.Cookies {
         if re.MatchString(c.Value) {
            matched = true
            break
         }
      }

      // 2. Check Response Headers
      if !matched {
         for _, h := range entry.Response.Headers {
            if re.MatchString(h.Value) {
               matched = true
               break
            }
         }
      }

      // 3. Check Response Body Text
      if !matched {
         if re.MatchString(entry.Response.Content.Text) {
            matched = true
         }
      }

      // If we found a match in any of the above, keep the entire raw entry
      if matched {
         keptEntries = append(keptEntries, rawEntry)
      }
   }

   har.Log.Entries = keptEntries

   outData, err := json.MarshalIndent(har, "", "  ")
   if err != nil {
      return fmt.Errorf("error marshalling output: %w", err)
   }

   if err := os.WriteFile(outputFile, outData, 0644); err != nil {
      return fmt.Errorf("error writing to output file: %w", err)
   }

   fmt.Printf("Success! Kept %d matching requests. Saved to %s\n", len(keptEntries), outputFile)
   return nil
}
