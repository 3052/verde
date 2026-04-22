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
   // Ensure an input file was provided
   if len(os.Args) < 2 {
      fmt.Printf("Usage: go run %s <input_file.txt>\n", filepath.Base(os.Args[0]))
      os.Exit(1)
   }

   inputFilePath := os.Args[1]
   inputFile, err := os.Open(inputFilePath)
   if err != nil {
      log.Fatalf("Failed to open input file: %v", err)
   }
   defer inputFile.Close()

   var currentFile *os.File
   // Ensure we close the last file if the script exits unexpectedly
   defer func() {
      if currentFile != nil {
         currentFile.Close()
      }
   }()

   scanner := bufio.NewScanner(inputFile)
   startMarker := "// --- START OF FILE "
   endMarker := "// --- END OF FILE "
   markerSuffix := " ---"

   filesCreated := 0

   for scanner.Scan() {
      line := scanner.Text()

      // Detect the START marker
      if strings.HasPrefix(line, startMarker) && strings.HasSuffix(line, markerSuffix) {
         // If a file is already open, close it before opening a new one
         if currentFile != nil {
            currentFile.Close()
            currentFile = nil
         }

         // Extract the filename from the marker
         filename := strings.TrimSuffix(strings.TrimPrefix(line, startMarker), markerSuffix)

         // Ensure the directory structure exists (just in case the filename includes folders)
         if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
            log.Fatalf("Failed to create directories for %s: %v", filename, err)
         }

         // Create the file
         currentFile, err = os.Create(filename)
         if err != nil {
            log.Fatalf("Failed to create file %s: %v", filename, err)
         }

         fmt.Printf("Extracting: %s...\n", filename)
         filesCreated++
         continue
      }

      // Detect the END marker
      if strings.HasPrefix(line, endMarker) && strings.HasSuffix(line, markerSuffix) {
         if currentFile != nil {
            currentFile.Close()
            currentFile = nil
         }
         continue
      }

      // Write the line to the current file (if we are inside a START/END block)
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
