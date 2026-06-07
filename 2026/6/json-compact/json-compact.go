package main

import (
   "bytes"
   "encoding/json"
   "flag"
   "fmt"
   "log"
   "os"
)

func main() {
   // 1. Define command-line flags
   inputFile := flag.String("i", "", "Path to the input JSON file (required)")
   outputFile := flag.String("o", "", "Path to the output JSON file (required)")
   
   // 2. Parse the flags
   flag.Parse()
   
   // 3. Ensure both flags were provided
   if *inputFile == "" || *outputFile == "" {
      flag.Usage()
      os.Exit(1)
   }

   // Delegate the work and handle any resulting errors at the entry point
   if err := compactJSONFile(*inputFile, *outputFile); err != nil {
      log.Fatal(err)
   }
}

func compactJSONFile(inFile, outFile string) error {
   // 4. Read the input file
   inputJSON, err := os.ReadFile(inFile)
   if err != nil {
      return fmt.Errorf("failed to read input file: %w", err)
   }
   
   // 5. Create a buffer and compact the JSON
   var compacted bytes.Buffer
   err = json.Compact(&compacted, inputJSON)
   if err != nil {
      return fmt.Errorf("invalid JSON input: %w", err)
   }
   
   // 6. Write the compacted JSON to the output file using os.ModePerm
   err = os.WriteFile(outFile, compacted.Bytes(), os.ModePerm)
   if err != nil {
      return fmt.Errorf("failed to write to output file: %w", err)
   }
   
   // 7. Log success with file sizes using the log package
   log.Printf("Successfully compacted '%s' into '%s'\n", inFile, outFile)
   log.Printf("Input size:  %d bytes\n", len(inputJSON))
   log.Printf("Output size: %d bytes\n", compacted.Len())

   return nil
}
