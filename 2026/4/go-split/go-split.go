package main

import (
   "bufio"
   "fmt"
   "log"
   "os"
   "path/filepath"
   "strings"
)

func main() {
   if len(os.Args) < 2 {
      fmt.Printf("Usage: %s <input_file.txt>\n", filepath.Base(os.Args[0]))
      os.Exit(1)
   }

   inputFile, err := os.Open(os.Args[1])
   if err != nil {
      log.Fatalf("Failed to open input file: %v", err)
   }
   defer inputFile.Close()

   scanner := bufio.NewScanner(inputFile)

   startMarker := "// FILE: "

   var currentFilename string
   var content strings.Builder
   filesCreated := 0

   // Helper function to flush the buffer to disk
   saveCurrentFile := func() {
      if currentFilename == "" {
         return
      }
      if err := os.MkdirAll(filepath.Dir(currentFilename), 0755); err != nil {
         log.Fatalf("Failed to create dirs for %s: %v", currentFilename, err)
      }
      if err := os.WriteFile(currentFilename, []byte(content.String()), 0644); err != nil {
         log.Fatalf("Failed to write %s: %v", currentFilename, err)
      }
      fmt.Printf("Extracting: %s...\n", currentFilename)
      filesCreated++
      content.Reset()
   }

   for scanner.Scan() {
      line := scanner.Text()

      // Detect the start of a new file
      if strings.HasPrefix(line, startMarker) {
         saveCurrentFile() // Save previous file

         currentFilename = strings.TrimPrefix(line, startMarker)
         currentFilename = strings.TrimSpace(currentFilename)
         continue
      }

      if currentFilename != "" {
         content.WriteString(line + "\n")
      }
   }

   saveCurrentFile() // Save the final file

   if err := scanner.Err(); err != nil {
      log.Fatalf("Error reading the input file: %v", err)
   }

   fmt.Printf("\nDone! Successfully extracted %d files.\n", filesCreated)
}
