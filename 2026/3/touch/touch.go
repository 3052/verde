package main

import (
   "log"
   "os"
   "time"
)

func main() {
   now := time.Now()
   for _, arg := range os.Args[1:] {
      err := os.Chtimes(arg, now, now)
      if err != nil {
         log.Fatal(err)
      }
   }
}
