package main

import (
   "bufio"
   "fmt"
   "os"
   "path/filepath"
   "sort"
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
   if len(os.Args) < 2 {
      fmt.Println("Usage: go run main.go <root_dir> [output.md]")
      os.Exit(1)
   }
   root := os.Args[1]
   outPath := "library.md"
   if len(os.Args) >= 3 {
      outPath = os.Args[2]
   }

   entries, errs := scan(root)
   for _, e := range errs {
      fmt.Fprintln(os.Stderr, "ERROR:", e)
   }
   if len(errs) > 0 {
      os.Exit(1)
   }

   if err := writeMarkdown(outPath, entries); err != nil {
      fmt.Fprintln(os.Stderr, "ERROR writing output:", err)
      os.Exit(1)
   }
   fmt.Println("Wrote", outPath, "with", len(entries), "entries")
}

// readURL reads the file and returns the first line that looks like a URL.
func readURL(path string) (string, error) {
   f, err := os.Open(path)
   if err != nil {
      return "", err
   }
   defer f.Close()

   sc := bufio.NewScanner(f)
   for sc.Scan() {
      line := strings.TrimSpace(sc.Text())
      if line == "" {
         continue
      }
      if strings.HasPrefix(line, "http://") || strings.HasPrefix(line, "https://") {
         return line, nil
      }
   }
   if err := sc.Err(); err != nil {
      return "", err
   }
   return "", nil
}

// writeMarkdown writes the library grouped by year descending,
// sorted alphabetically by URL within each year.
func writeMarkdown(path string, entries []Entry) error {
   byYear := map[string][]Entry{}
   for _, e := range entries {
      byYear[e.Year] = append(byYear[e.Year], e)
   }

   years := make([]string, 0, len(byYear))
   for y := range byYear {
      years = append(years, y)
   }
   sort.Sort(sort.Reverse(sort.StringSlice(years)))

   var b strings.Builder
   b.WriteString("# library\n\n")

   for i, y := range years {
      if i > 0 {
         b.WriteString("\n")
      }
      b.WriteString("## " + y + "\n\n")

      items := byYear[y]
      sort.Slice(items, func(i, j int) bool {
         return items[i].URL < items[j].URL
      })

      if len(items) == 1 {
         b.WriteString(items[0].URL + "\n")
      } else {
         for _, e := range items {
            b.WriteString("- " + e.URL + "\n")
         }
      }
   }

   return os.WriteFile(path, []byte(b.String()), 0o644)
}

// Entry represents one movie folder + its URL
type Entry struct {
   Folder string
   Year   string
   URL    string
}

// scan walks root, finds each "Name (Year)/readme.txt", validates it,
// and reads the URL from the first non-empty line.
func scan(root string) ([]Entry, []string) {
   var entries []Entry
   var errs []string

   rootAbs, err := filepath.Abs(root)
   if err != nil {
      return nil, []string{fmt.Sprintf("cannot resolve root: %v", err)}
   }
   _ = rootAbs

   dirEnts, err := os.ReadDir(root)
   if err != nil {
      return nil, []string{fmt.Sprintf("cannot read root dir: %v", err)}
   }

   for _, de := range dirEnts {
      if !de.IsDir() {
         continue
      }
      name := de.Name()

      year, ok := extractYear(name)
      if !ok {
         errs = append(errs, fmt.Sprintf("no year in folder name: %q", name))
         continue
      }

      readmePath := filepath.Join(root, name, "readme.txt")

      info, err := os.Stat(readmePath)
      if err != nil {
         errs = append(errs, fmt.Sprintf("missing readme.txt in %q: %v", name, err))
         continue
      }
      if info.Size() == 0 {
         errs = append(errs, fmt.Sprintf("empty readme.txt in %q", name))
         continue
      }

      url, err := readURL(readmePath)
      if err != nil {
         errs = append(errs, fmt.Sprintf("cannot read URL from %q: %v", readmePath, err))
         continue
      }
      if url == "" {
         errs = append(errs, fmt.Sprintf("no URL found in %q", readmePath))
         continue
      }

      entries = append(entries, Entry{Folder: name, Year: year, URL: url})
   }

   return entries, errs
}
