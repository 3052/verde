package main

import (
   "bytes"
   "encoding/json"
   "flag"
   "fmt"
   "io/fs"
   "os"
   "os/exec"
   "path/filepath"
   "strconv"
   "strings"
)

// Constants for file sizes and limits
const (
   KB = 1024
   MB = 1024 * KB
   GB = 1024 * MB

   SizeThreshold = 4 * int64(GB)
)

var allowedExtensions = map[string]bool{
   ".mp4":  true,
   ".m4a":  true,
   ".ini":  true,
   ".jpg":  true,
   ".vtt":  true,
   ".ts":   true,
   ".txt":  true,
   ".json": true, // Added so metadata.json isn't flagged
}

// mediaExtensions defines which files should be checked for bitrate.
var mediaExtensions = map[string]bool{
   ".mp4": true,
   ".ts":  true,
}

// getBitrate uses ffprobe to fetch the overall bitrate of a media file.
func getBitrate(path string) (int64, error) {
   cmd := exec.Command("ffprobe",
      "-show_entries", "format=bit_rate",
      "-of", "default=noprint_wrappers=1:nokey=1",
      path,
   )

   var out bytes.Buffer
   cmd.Stdout = &out
   // Discard stderr to prevent ffprobe banner and logs from spamming the terminal
   cmd.Stderr = nil

   if err := cmd.Run(); err != nil {
      return 0, err
   }

   // ParseInt will return an error for "N/A" or empty strings.
   return strconv.ParseInt(strings.TrimSpace(out.String()), 10, 64)
}

func main() {
   var rootDir string
   var minBitrate int64

   flag.StringVar(&rootDir, "dir", "", "Root directory to audit")
   flag.Int64Var(&minBitrate, "min-bitrate", 2_400_000, "Minimum acceptable bitrate in bits per second")
   flag.Parse()

   // Require the user to provide a directory via the flag
   if rootDir == "" {
      flag.Usage()
      os.Exit(1)
   }

   fmt.Printf("Auditing '%s' (Minimum Bitrate: %d bps)...\n", rootDir, minBitrate)
   fmt.Println(strings.Repeat("-", 70))

   auditor := &Auditor{
      MinBitrate: minBitrate,
   }
   err := filepath.WalkDir(rootDir, auditor.auditFile)

   if err != nil {
      fmt.Fprintf(os.Stderr, "Error walking the directory: %v\n", err)
   }

   fmt.Println(strings.Repeat("-", 70))

   // --- PRINT EXCLUDES AT THE END ---
   if len(auditor.ExcludedDirs) > 0 {
      fmt.Println("Excluded Folders:")
      for _, ex := range auditor.ExcludedDirs {
         fmt.Printf("[Skipped] %s\n", ex)
      }
      fmt.Println(strings.Repeat("-", 70))
   }

   // --- PRINT FAILURES AT THE END ---
   if len(auditor.Failures) > 0 {
      fmt.Println("Failures:")
      for _, fail := range auditor.Failures {
         fmt.Printf("[%s] %s\n", strings.Join(fail.Flags, ", "), fail.Path)
      }
      fmt.Println(strings.Repeat("-", 70))
   }

   fmt.Println("Audit complete.")
}

// AuditResult stores the path and the reasons why a file failed the audit.
type AuditResult struct {
   Path  string
   Flags []string
}

// Auditor holds the state of the audit, including all failures and exclusions.
type Auditor struct {
   Failures     []AuditResult
   ExcludedDirs []string
   MinBitrate   int64
}

// auditFile is the WalkDir callback that flags files exceeding the size
// threshold, carrying an unexpected extension, or having too low bitrate.
func (a *Auditor) auditFile(path string, entry fs.DirEntry, err error) error {
   if err != nil {
      return err
   }

   if entry.IsDir() {
      // Check for metadata.json in this directory
      metaPath := filepath.Join(path, "metadata.json")
      data, err := os.ReadFile(metaPath)
      if err == nil {
         // File exists, check if "exclude" is true
         var meta struct {
            Exclude bool `json:"exclude"`
         }
         if err := json.Unmarshal(data, &meta); err == nil {
            if meta.Exclude {
               fmt.Printf("Skipping excluded folder: %s\n", path)
               // Record the excluded directory for the end summary
               a.ExcludedDirs = append(a.ExcludedDirs, path)
               return filepath.SkipDir // Bypasses the entire folder
            }
         }
      }
      return nil
   }

   info, err := entry.Info()
   if err != nil {
      return err
   }

   // --- PRINT NAME DURING WALK ---
   fmt.Println(path)

   var flags []string

   // --- RULE 1: Size Check ---
   if info.Size() > SizeThreshold {
      sizeInGB := float64(info.Size()) / float64(GB)
      flags = append(flags, fmt.Sprintf("%.2f GB", sizeInGB))
   }

   // --- RULE 2: Extension Check ---
   ext := strings.ToLower(filepath.Ext(path))
   if !allowedExtensions[ext] {
      if ext == "" {
         flags = append(flags, "No Extension")
      } else {
         flags = append(flags, fmt.Sprintf("Ext: %s", ext))
      }
   }

   // --- RULE 3: Bitrate Check (for media files) ---
   if mediaExtensions[ext] {
      bitrate, err := getBitrate(path)
      if err != nil {
         // Propagate the error to halt the walk and report it
         return fmt.Errorf("failed to get bitrate for %s: %w", path, err)
      }

      // Use the struct's MinBitrate field instead of the hardcoded constant
      if bitrate > 0 && bitrate < a.MinBitrate {
         kbps := bitrate / 1000
         flags = append(flags, fmt.Sprintf("%d kbps", kbps))
      }
   }

   // --- COLLECT FAILURES ---
   if len(flags) > 0 {
      a.Failures = append(a.Failures, AuditResult{Path: path, Flags: flags})
   }

   return nil
}
