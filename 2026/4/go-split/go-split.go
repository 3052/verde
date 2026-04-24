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

   var currentFile *os.File
   defer func() {
      if currentFile != nil {
         currentFile.Close()
      }
   }()

   scanner := bufio.NewScanner(inputFile)

   // The simplest, valid-Go-syntax marker style
   startMarker := "// --- START OF FILE "
   endMarker := "// --- END OF FILE "
   markerSuffix := " ---"

   filesCreated := 0

   for scanner.Scan() {
      line := scanner.Text()

      // Detect START marker
      if strings.HasPrefix(line, startMarker) && strings.HasSuffix(line, markerSuffix) {
         if currentFile != nil {
            currentFile.Close()
            currentFile = nil
         }

         filename := strings.TrimPrefix(line, startMarker)
         filename = strings.TrimSuffix(filename, markerSuffix)
         filename = strings.TrimSpace(filename)

         if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
            log.Fatalf("Failed to create directories for %s: %v", filename, err)
         }

         f, err := os.Create(filename)
         if err != nil {
            log.Fatalf("Failed to create file %s: %v", filename, err)
         }
         currentFile = f

         fmt.Printf("Extracting: %s...\n", filename)
         filesCreated++
         continue
      }

      // Detect END marker
      if strings.HasPrefix(line, endMarker) && strings.HasSuffix(line, markerSuffix) {
         if currentFile != nil {
            currentFile.Close()
            currentFile = nil
         }
         continue
      }

      // Write the line to the current file
      if currentFile != nil {
         if _, err := currentFile.WriteString(line + "\n"); err != nil {
            log.Fatalf("Failed to write to file: %v", err)
         }
      }
   }

   if err := scanner.Err(); err != nil {
      log.Fatalf("Error reading the input file: %v", err)
   }

   fmt.Printf("\nDone! Successfully extracted %d files.\n", filesCreated)
}
