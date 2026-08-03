package main

import (
   "encoding/json"
   "flag"
   "fmt"
   "os"
   "path/filepath"
   "slices"
   "strings"
)

// extractYear manually finds the year inside the last set of parentheses.
func extractYear(name string) (string, bool) {
   start := strings.LastIndex(name, "(")
   end := strings.LastIndex(name, ")")

   if start == -1 || end == -1 || end < start {
      return "", false
   }

   year := name[start+1 : end]
   if len(year) == 4 {
      return year, true
   }

   return "", false
}

func main() {
   root := flag.String("root", "", "root directory containing movie folders")
   outPath := flag.String("out", "library.md", "output markdown file path")
   flag.Parse()

   if *root == "" {
      fmt.Fprintln(os.Stderr, "Usage: go run main.go -root <root_dir> [-out output.md]")
      flag.PrintDefaults()
      os.Exit(1)
   }

   entries, errs := scan(*root)
   for _, msg := range errs {
      fmt.Fprintln(os.Stderr, "ERROR:", msg)
   }
   if len(errs) > 0 {
      os.Exit(1)
   }

   if err := writeMarkdown(*outPath, entries); err != nil {
      fmt.Fprintln(os.Stderr, "ERROR writing output:", err)
      os.Exit(1)
   }
   fmt.Println("Wrote", *outPath, "with", len(entries), "entries")
}

// readURL reads the file as JSON and returns the "url" field.
func readURL(path string) (string, error) {
   data, err := os.ReadFile(path)
   if err != nil {
      return "", err
   }

   var doc struct {
      URL string `json:"url"`
   }
   if err := json.Unmarshal(data, &doc); err != nil {
      return "", err
   }
   if doc.URL == "" {
      return "", fmt.Errorf("no url field in %s", path)
   }
   return doc.URL, nil
}

// writeMarkdown writes the library grouped by year descending,
// sorted alphabetically by URL within each year.
func writeMarkdown(path string, entries []Entry) error {
   byYear := map[string][]Entry{}
   for _, entry := range entries {
      byYear[entry.Year] = append(byYear[entry.Year], entry)
   }

   years := make([]string, 0, len(byYear))
   for year := range byYear {
      years = append(years, year)
   }
   slices.SortFunc(years, func(first, second string) int {
      return strings.Compare(second, first) // descending
   })

   var builder strings.Builder
   builder.WriteString("# library\n\n")

   for i, year := range years {
      if i > 0 {
         builder.WriteString("\n")
      }
      builder.WriteString("## " + year + "\n\n")

      items := byYear[year]
      slices.SortFunc(items, func(first, second Entry) int {
         return strings.Compare(first.URL, second.URL) // ascending
      })

      if len(items) == 1 {
         builder.WriteString(items[0].URL + "\n")
      } else {
         for _, entry := range items {
            builder.WriteString("- " + entry.URL + "\n")
         }
      }
   }

   return os.WriteFile(path, []byte(builder.String()), 0o644)
}

// Entry represents one movie folder + its URL
type Entry struct {
   Folder string
   Year   string
   URL    string
}

// scan walks root, finds each "Name (Year)/metadata.json", validates it,
// and reads the URL from the JSON "url" field.
func scan(root string) ([]Entry, []string) {
   var entries []Entry
   var errs []string

   dirEnts, err := os.ReadDir(root)
   if err != nil {
      return nil, []string{fmt.Sprintf("cannot read root dir: %v", err)}
   }

   for _, dirEnt := range dirEnts {
      if !dirEnt.IsDir() {
         continue
      }
      name := dirEnt.Name()

      year, ok := extractYear(name)
      if !ok {
         errs = append(errs, fmt.Sprintf("no year in folder name: %q", name))
         continue
      }

      metadataPath := filepath.Join(root, name, "metadata.json")

      info, err := os.Stat(metadataPath)
      if err != nil {
         errs = append(errs, fmt.Sprintf("missing metadata.json in %q: %v", name, err))
         continue
      }
      if info.Size() == 0 {
         errs = append(errs, fmt.Sprintf("empty metadata.json in %q", name))
         continue
      }

      url, err := readURL(metadataPath)
      if err != nil {
         errs = append(errs, fmt.Sprintf("cannot read URL from %q: %v", metadataPath, err))
         continue
      }

      entries = append(entries, Entry{Folder: name, Year: year, URL: url})
   }

   return entries, errs
}
