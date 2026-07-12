package main

import (
   "bytes"
   "encoding/json"
   "flag"
   "fmt"
   "mime"
   "os"
   "sort"
   "strings"
)

var defaultExcludeTypes = []string{
   "<missing>",
   "application/binary",
   "application/dash+xml",
   "application/font-woff",
   "application/font-woff2",
   "application/javascript",
   "application/octet-stream",
   "application/x-font-ttf",
   "application/x-javascript",
   "application/x-protobuf",
   "binary/octet-stream",
   "font/ttf",
   "font/woff2",
   "image/avif",
   "image/gif",
   "image/jpeg",
   "image/png",
   "image/svg+xml",
   "image/webp",
   "image/x-icon",
   "text/css",
   "text/event-stream",
   "text/javascript",
   "text/plain",
   "text/plain charset=utf-8",
   "text/xml",
   "video/mp4",
}

func main() {
   var inputFile, excludeTypes string

   defaultExcludeStr := strings.Join(defaultExcludeTypes, ",")

   flag.StringVar(&inputFile, "i", "", "Path to the input .har file (required)")
   flag.StringVar(&excludeTypes, "c", defaultExcludeStr, "Comma-separated list of Content-Types to remove")

   flag.Parse()

   if inputFile == "" {
      fmt.Fprintln(os.Stderr, "Error: Missing required argument (-i).")
      flag.PrintDefaults()
      os.Exit(1)
   }

   if err := processHAR(inputFile, excludeTypes); err != nil {
      fmt.Fprintf(os.Stderr, "Fatal error: %v\n", err)
      os.Exit(1)
   }
}

func processHAR(inputFile, excludeTypes string) error {
   excludeMap := make(map[string]bool)
   for _, t := range strings.Split(excludeTypes, ",") {
      cleanType := strings.ToLower(strings.TrimSpace(t))
      if cleanType != "" {
         excludeMap[cleanType] = true
      }
   }

   outputFile := "output.har"

   data, err := os.ReadFile(inputFile)
   if err != nil {
      return fmt.Errorf("error reading file: %w", err)
   }

   inputSize := len(data)

   var har HAR
   if err := json.Unmarshal(data, &har); err != nil {
      return fmt.Errorf("error parsing HAR JSON: %w", err)
   }

   inputCount := len(har.Log.Entries)

   fmt.Printf("Input size: %d bytes\n", inputSize)

   var keptEntries []json.RawMessage

   // Map to track the total byte size of entries per content type
   contentTypeSizes := make(map[string]int)

   for i, rawEntry := range har.Log.Entries {
      var entry PartialEntry
      if err := json.Unmarshal(rawEntry, &entry); err != nil {
         fmt.Fprintf(os.Stderr, "Warning: failed to parse entry %d: %v\n", i, err)
         continue
      }

      shouldExclude := false
      currentMediaType := "<missing>"

      for _, h := range entry.Response.Headers {
         if strings.EqualFold(h.Name, "content-type") {
            mediaType, _, err := mime.ParseMediaType(h.Value)
            if err != nil {
               mediaType = strings.ToLower(strings.Split(h.Value, ";")[0])
            }

            currentMediaType = strings.TrimSpace(mediaType)
            break
         }
      }

      // Check the exclusion map AFTER the loop.
      // This ensures that if it remained "<missing>", it can still be excluded.
      if excludeMap[currentMediaType] {
         shouldExclude = true
      }

      if !shouldExclude {
         keptEntries = append(keptEntries, rawEntry)

         // Compact the raw JSON to strip formatting whitespace before measuring size
         var compacted bytes.Buffer
         if err := json.Compact(&compacted, rawEntry); err != nil {
            return fmt.Errorf("error compacting JSON for entry %d: %w", i, err)
         }

         contentTypeSizes[currentMediaType] += compacted.Len()
      }
   }

   har.Log.Entries = keptEntries

   // Marshal without indent to keep the output compact
   outData, err := json.Marshal(har)
   if err != nil {
      return fmt.Errorf("error marshalling output: %w", err)
   }

   outputSize := len(outData)

   if err := os.WriteFile(outputFile, outData, 0644); err != nil {
      return fmt.Errorf("error writing to output file: %w", err)
   }

   fmt.Printf("Output size: %d bytes\n", outputSize)
   fmt.Printf("Success! Kept %d requests (removed %d). Saved to %s\n", len(keptEntries), inputCount-len(keptEntries), outputFile)

   fmt.Println("\nOutput Content-Types (by size):")

   type ctSize struct {
      name string
      size int
   }

   var sortedTypes []ctSize
   for t, size := range contentTypeSizes {
      sortedTypes = append(sortedTypes, ctSize{name: t, size: size})
   }

   // Sort descending by size
   sort.Slice(sortedTypes, func(i, j int) bool {
      return sortedTypes[i].size > sortedTypes[j].size
   })

   for _, t := range sortedTypes {
      fmt.Printf("  - %s: %d bytes\n", t.name, t.size)
   }

   return nil
}

type HAR struct {
   Log struct {
      Version string            `json:"version"`
      Creator interface{}       `json:"creator"`
      Pages   interface{}       `json:"pages,omitempty"`
      Entries []json.RawMessage `json:"entries"`
   } `json:"log"`
}

type NameValuePair struct {
   Name  string `json:"name"`
   Value string `json:"value"`
}

type PartialEntry struct {
   Response struct {
      Headers []NameValuePair `json:"headers"`
   } `json:"response"`
}
