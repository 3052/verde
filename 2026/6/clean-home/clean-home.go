package main

import (
   "fmt"
   "log"
   "os"
   "path/filepath"
)

var patterns = []string{
   `C:\ProgramData\Mozilla-*`,
   `C:\ProgramData\Mozilla`,
   `C:\Users\Steven\.android`,
   `C:\Users\Steven\.cargo`,
   `C:\Users\Steven\AppData\Local\Android Open Source Project`,
   `C:\Users\Steven\AppData\Local\Android\Sdk\system-images\android-*`,
   `C:\Users\Steven\AppData\Local\Genymobile`,
   `C:\Users\Steven\AppData\Local\Google`,
   `C:\Users\Steven\AppData\Local\Mozilla\Firefox\Profiles\*.*\cache2`,
   `C:\Users\Steven\AppData\Local\go-build`,
   `C:\Users\Steven\AppData\Local\pip`,
   `C:\Users\Steven\go\pkg`,
}

func main() {
   for _, pattern := range patterns {
      matches, err := filepath.Glob(pattern)
      if err != nil {
         log.Fatal(err)
      }
      for _, match := range matches {
         fmt.Println(match)
         err := os.RemoveAll(match)
         if err != nil {
            log.Fatal(err)
         }
      }
   }
   log.SetFlags(log.Ltime)
   log.Print("disk cleanup")
   log.Print("recycle bin")
   log.Print(`C:\Users\Steven\AppData\Local\Microsoft\Edge`)
   log.Print(`C:\Users\Steven\AppData\Roaming\Mozilla\Firefox\Profiles\b5ohm1qd.2024-11-28`)
   log.Print(`C:\Users\Steven\AppData\Roaming\Mozilla\Firefox\Profiles\uuivvd0h.2024-11-29`)
}
