package main

import (
   "flag"
   "fmt"
   "log"
   "os"
   "path/filepath"
   "strings"
)

func main() {
   // Keep CLI output clean by removing timestamp prefixes from standard log output
   log.SetFlags(0)

   writeFlag := flag.Bool("w", false, "Write result to the source file instead of stdout")
   flag.Parse()

   if flag.NArg() < 1 {
      flag.Usage()
      os.Exit(1)
   }

   targetDir := flag.Arg(0)

   if err := processDirectory(targetDir, *writeFlag); err != nil {
      log.Fatal(err)
   }
}

// processDirectory enforces that the target is a directory and
// routes all .go files to the processor.
func processDirectory(targetDir string, writeResult bool) error {
   info, err := os.Stat(targetDir)
   if err != nil {
      return fmt.Errorf("error accessing path: %w", err)
   }

   if !info.IsDir() {
      return fmt.Errorf("error: target must be a directory, not a file")
   }

   // Walk the directory and process all .go files
   err = filepath.WalkDir(targetDir, func(path string, d os.DirEntry, err error) error {
      if err != nil {
         return err // Bubble up access/read errors immediately
      }
      if !d.IsDir() && strings.HasSuffix(path, ".go") {
         if err := processFile(path, writeResult); err != nil {
            // Return the error to halt the WalkDir process immediately
            return fmt.Errorf("error processing %s: %w", path, err)
         }
      }
      return nil
   })

   if err != nil {
      return fmt.Errorf("error walking directory: %w", err)
   }

   return nil
}
