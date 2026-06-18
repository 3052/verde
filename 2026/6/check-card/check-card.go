package main

import (
   "bufio"
   "cmp"
   "encoding/csv"
   "flag"
   "fmt"
   "os"
   "regexp"
   "slices"
   "strconv"
   "strings"
   "time"
)

var targets = []string{
   "BRAUMS STORE",
   "CHICK-FIL-A",
   "JASON'S DELI",
   "LA MADELEINE",
   "MCDONALD'S",
   "SCHLOTZSKYS",
   "SPRING CREEK",
   "WENDY",
   "WHATABURGER",
}

// MatchResult holds a Count and the target Description
type MatchResult struct {
   Description string
   Count       int
}

func main() {
   // Define command-line flag for the input file
   inputFile := flag.String("f", "", "Path to the bank statement text file")
   flag.Parse()

   // Ensure the file flag was provided
   if *inputFile == "" {
      flag.Usage()
      os.Exit(1)
   }

   now := time.Now()
   today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

   // 1. Extract fallible reading logic
   validDescriptions, err := extractRecentDescriptions(*inputFile, today, 7)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Fatal error reading descriptions: %v\n", err)
      os.Exit(1)
   }

   // 2. Count occurrences (no fallible logic here)
   var results []*MatchResult
   for _, target := range targets {
      count := 0
      for _, desc := range validDescriptions {
         if strings.Contains(desc, target) {
            count++
         }
      }
      results = append(results, &MatchResult{
         Description: target,
         Count:       count,
      })
   }

   // 3. Sort by Count descending
   slices.SortFunc(results, func(a, b *MatchResult) int {
      return cmp.Compare(a.Count, b.Count)
   })

   // 4. Extract fallible writing logic
   if err := writeCSV(results); err != nil {
      fmt.Fprintf(os.Stderr, "Fatal error writing CSV: %v\n", err)
      os.Exit(1)
   }
}

// extractRecentDescriptions handles file I/O, regex, date parsing, and filtering.
func extractRecentDescriptions(filename string, today time.Time, lookbackDays int) ([]string, error) {
   file, err := os.Open(filename)
   if err != nil {
      return nil, fmt.Errorf("failed to open file: %w", err)
   }
   defer file.Close()

   cutoffDate := today.AddDate(0, 0, -lookbackDays)
   var validDescriptions []string
   scanner := bufio.NewScanner(file)

   dateRegex := regexp.MustCompile(`\b(\d{2}/\d{2})\b`)

   for scanner.Scan() {
      line := strings.TrimSpace(scanner.Text())

      if line == "Description" {
         if scanner.Scan() {
            desc := strings.TrimSpace(scanner.Text())

            match := dateRegex.FindString(desc)
            if match != "" {
               parsedMMDD, err := time.Parse("01/02", match)
               if err == nil {
                  year := today.Year()

                  if parsedMMDD.Month() > today.Month() {
                     year--
                  }

                  parsedDate := time.Date(year, parsedMMDD.Month(), parsedMMDD.Day(), 0, 0, 0, 0, today.Location())

                  if parsedDate.After(cutoffDate) {
                     validDescriptions = append(validDescriptions, desc)
                  }

               }
            }
         }
      }
   }

   if err := scanner.Err(); err != nil {
      return nil, fmt.Errorf("error reading file: %w", err)
   }

   return validDescriptions, nil
}

// writeCSV safely writes the sorted results to standard output, returning any encoding errors.
func writeCSV(results []*MatchResult) error {
   writer := csv.NewWriter(os.Stdout)

   if err := writer.Write([]string{"count", "description"}); err != nil {
      return fmt.Errorf("failed to write csv header: %w", err)
   }

   for _, res := range results {
      row := []string{
         strconv.Itoa(res.Count),
         res.Description,
      }
      if err := writer.Write(row); err != nil {
         return fmt.Errorf("failed to write csv row for %s: %w", res.Description, err)
      }
   }

   writer.Flush()
   if err := writer.Error(); err != nil {
      return fmt.Errorf("failed to flush csv writer: %w", err)
   }

   return nil
}
