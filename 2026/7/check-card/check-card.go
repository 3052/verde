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
   "ROMANO'S PIZ",
   "SCHLOTZSKYS",
   "SHAWARMA PRESS",
   "SPRING CREEK",
   "STARBUCKS",
   "WENDY",
   "WHATABURGER",
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
   validTransactions, err := extractRecentTransactions(*inputFile, today, 7)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Fatal error reading descriptions: %v\n", err)
      os.Exit(1)
   }

   // 2. Count occurrences and track the latest date
   var results []*MatchResult
   for _, target := range targets {
      count := 0
      var lastDate time.Time

      for _, txn := range validTransactions {
         if strings.Contains(txn.Description, target) {
            count++
            // Update lastDate if this transaction is more recent
            if txn.Date.After(lastDate) {
               lastDate = txn.Date
            }
         }
      }

      results = append(results, &MatchResult{
         Description: target,
         Count:       count,
         LastDate:    lastDate,
      })
   }

   // 3. Sort by Count (ascending) THEN Date (ascending)
   slices.SortFunc(results, func(a, b *MatchResult) int {
      // First compare by Count (a vs b for ascending order)
      if countDiff := cmp.Compare(a.Count, b.Count); countDiff != 0 {
         return countDiff
      }
      // If counts are equal, compare by LastDate (a vs b for ascending order)
      return a.LastDate.Compare(b.LastDate)
   })

   // 4. Extract fallible writing logic
   if err := writeCSV(results); err != nil {
      fmt.Fprintf(os.Stderr, "Fatal error writing CSV: %v\n", err)
      os.Exit(1)
   }
}

// writeCSV safely writes the sorted results to standard output, returning any encoding errors.
func writeCSV(results []*MatchResult) error {
   writer := csv.NewWriter(os.Stdout)

   if err := writer.Write([]string{"count", "description", "last_date"}); err != nil {
      return fmt.Errorf("failed to write csv header: %w", err)
   }

   for _, res := range results {
      // Format the date if matches were found, otherwise leave blank
      dateStr := ""
      if res.Count > 0 {
         dateStr = res.LastDate.Format("2006-01-02")
      }

      row := []string{
         strconv.Itoa(res.Count),
         res.Description,
         dateStr,
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

// MatchResult holds a Count, the target Description, and the most recent match date
type MatchResult struct {
   Description string
   Count       int
   LastDate    time.Time
}

// Transaction holds the parsed date along with the description
type Transaction struct {
   Description string
   Date        time.Time
}

// extractRecentTransactions handles file I/O, regex, date parsing, and filtering.
func extractRecentTransactions(filename string, today time.Time, lookbackDays int) ([]Transaction, error) {
   file, err := os.Open(filename)
   if err != nil {
      return nil, fmt.Errorf("failed to open file: %w", err)
   }
   defer file.Close()

   cutoffDate := today.AddDate(0, 0, -lookbackDays)
   var validTransactions []Transaction
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
                     // Store both the description and the parsed date
                     validTransactions = append(validTransactions, Transaction{
                        Description: desc,
                        Date:        parsedDate,
                     })
                  }

               }
            }
         }
      }
   }

   if err := scanner.Err(); err != nil {
      return nil, fmt.Errorf("error reading file: %w", err)
   }

   return validTransactions, nil
}
