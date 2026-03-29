package main

import (
   "flag"
   "log"
   "os"
   "strings"
)

var contains = []string{
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

func do_check(name string) error {
   data, err := read_file(name)
   if err != nil {
      return err
   }
   for _, contain := range contains {
      if strings.Contains(data, contain) {
         log.Println("Contains", contain)
      }
   }
   log.Print()
   for _, contain := range contains {
      if !strings.Contains(data, contain) {
         log.Println("!Contains", contain)
      }
   }
   return nil
}

func main() {
   log.SetFlags(log.Ltime)
   name := flag.String("n", "", "name")
   flag.Parse()
   if *name != "" {
      err := do_check(*name)
      if err != nil {
         log.Fatal(err)
      }
   } else {
      flag.Usage()
   }
}

func read_file(name string) (string, error) {
   data, err := os.ReadFile(name)
   if err != nil {
      return "", err
   }
   return string(data), nil
}
