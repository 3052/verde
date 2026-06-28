package main

import (
   "flag"
   "fmt"
   "log"
   "os"
   "path/filepath"
)

// createStructure handles the generation of the directories and the empty readme file.
func createStructure(name, year, monthDay string) error {
   var dirPath string

   // Flatten the path structure based on whether monthDay (-m) is provided
   if monthDay != "" {
      dirPath = filepath.Join(year, monthDay, name)
   } else {
      dirPath = filepath.Join(year, name)
   }

   if err := os.MkdirAll(dirPath, 0755); err != nil {
      return fmt.Errorf("failed to create directories: %w", err)
   }

   filePath := filepath.Join(dirPath, "readme.md")

   // Check if the file already exists
   if _, err := os.Stat(filePath); err == nil {
      log.Println("File already exists, skipping:", filepath.ToSlash(filePath))
      return nil
   }

   // Create the file only if it doesn't exist
   if err := os.WriteFile(filePath, nil, 0644); err != nil {
      return fmt.Errorf("failed to create file: %w", err)
   }

   log.Println("Successfully created:", filepath.ToSlash(filePath))
   return nil
}

func main() {
   n := flag.String("n", "", "Required: Name to include in the path (e.g., 'amazon')")
   y := flag.String("y", "", "Required: Year (e.g., '2026')")
   m := flag.String("m", "", "Optional: Month and day (e.g., '6-4')")

   flag.Parse()

   // Enforce the required flags
   if *n == "" || *y == "" {
      flag.Usage()
      log.Fatal("Error: both -n and -y flags are required.")
   }

   if err := createStructure(*n, *y, *m); err != nil {
      log.Fatal("Error:", err)
   }
}
