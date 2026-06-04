package main

import (
   "bytes"
   "go/format"
   "io/fs"
   "log"
   "os"
   "path/filepath"
)

func walk_dir(path string, _ fs.DirEntry, err error) error {
   if err != nil {
      return err
   }
   if filepath.Ext(path) != ".go" {
      return nil
   }
   log.Print(path)
   data, err := os.ReadFile(path)
   if err != nil {
      return err
   }
   // 1. newlines
   data = bytes.ReplaceAll(
      data, []byte("\n}\n"), []byte("\n}\n\n"),
   )
   // 2. gofmt
   data, err = format.Source(data)
   if err != nil {
      return err
   }
   // 3. tabs
   data = bytes.ReplaceAll(
      data, []byte{'\t'}, []byte("   "),
   )
   return os.WriteFile(path, data, os.ModePerm)
}

func main() {
   log.SetFlags(log.Ltime)
   err := filepath.WalkDir(".", walk_dir)
   if err != nil {
      log.Fatal(err)
   }
}
