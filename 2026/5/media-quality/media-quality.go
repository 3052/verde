package main

import (
   "fmt"
   "io/fs"
   "os"
   "path/filepath"
   "strings"
)

// Constants for file sizes
const (
   KB = 1024
   MB = 1024 * KB
   GB = 1024 * MB

   SizeThreshold = 4 * int64(GB)
)

// Define the extensions we consider "valid".
// Everything else will be flagged. Must be lowercase.
var allowedExtensions = map[string]bool{
   ".mp4": true,
   ".m4a": true,
   ".md":  true,
   ".ini": true, // often created by Windows or media scrapers
   ".jpg": true, // cover art / thumbnails
   ".vtt": true, // subtitle files
}

func main() {
   rootDir := "."
   if len(os.Args) > 1 {
      rootDir = os.Args[1]
   }

   fmt.Printf("Auditing '%s'...\n", rootDir)
   fmt.Println(strings.Repeat("-", 70))

   err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
      if err != nil {
         return nil
      }

      if !d.IsDir() {
         info, err := d.Info()
         if err != nil {
            return nil
         }

         // We will collect any "violations" for this file in a slice (list)
         var flags []string

         // --- RULE 1: Size Check ---
         if info.Size() > SizeThreshold {
            sizeInGB := float64(info.Size()) / float64(GB)
            flags = append(flags, fmt.Sprintf("%.2f GB", sizeInGB))
         }

         // --- RULE 2: Extension Check ---
         // filepath.Ext gets the extension (including the dot).
         // strings.ToLower ensures ".JPG" matches ".jpg".
         ext := strings.ToLower(filepath.Ext(path))

         // If the extension is NOT in our map, flag it
         if !allowedExtensions[ext] {
            if ext == "" {
               flags = append(flags, "No Extension")
            } else {
               flags = append(flags, fmt.Sprintf("Ext: %s", ext))
            }
         }

         // --- REPORTING ---
         // If the file triggered any flags, print it out
         if len(flags) > 0 {
            flagString := strings.Join(flags, ", ")
            fmt.Printf("[%s] %s\n", flagString, path)
         }
      }

      return nil
   })

   if err != nil {
      fmt.Fprintf(os.Stderr, "Error walking the directory: %v\n", err)
   }

   fmt.Println(strings.Repeat("-", 70))
   fmt.Println("Audit complete.")
}
