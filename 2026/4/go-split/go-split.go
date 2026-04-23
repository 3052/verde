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
      fmt.Printf("Usage: %s <input_file.txt>\n", filepath.Base(os.Args[0]))
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

   // Supported start and end marker prefixes
   startMarker1 := "// --- START OF FILE "
   startMarker2 := "--- START OF FILE "
   endMarker1 := "// --- END OF FILE "
   endMarker2 := "--- END OF FILE "
   markerSuffix := " ---"

   filesCreated := 0

   for scanner.Scan() {
      line := scanner.Text()

      isStart1 := strings.HasPrefix(line, startMarker1)
      isStart2 := strings.HasPrefix(line, startMarker2)

      // Detect the START marker
      if (isStart1 || isStart2) && strings.HasSuffix(line, markerSuffix) {
         if currentFile != nil {
            currentFile.Close()
            currentFile = nil
         }

         // Extract the filename from whichever marker matched
         var filename string
         if isStart1 {
            filename = strings.TrimPrefix(line, startMarker1)
         } else {
            filename = strings.TrimPrefix(line, startMarker2)
         }
         filename = strings.TrimSuffix(filename, markerSuffix)
         filename = strings.TrimSpace(filename)

         if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
            log.Fatalf("Failed to create directories for %s: %v", filename, err)
         }

         currentFile, err = os.Create(filename)
         if err != nil {
            log.Fatalf("Failed to create file %s: %v", filename, err)
         }

         fmt.Printf("Extracting: %s...\n", filename)
         filesCreated++
         continue
      }

      // Detect the END marker
      isEnd1 := strings.HasPrefix(line, endMarker1)
      isEnd2 := strings.HasPrefix(line, endMarker2)

      if (isEnd1 || isEnd2) && strings.HasSuffix(line, markerSuffix) {
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
