package main

import (
   "fmt"
   "log"
   "os"
   "os/exec"
)

const (
   invert = "\x1b[7m"
   reset  = "\x1b[m"
)

func main() {
   dirs, err := os.ReadDir(".")
   if err != nil {
      log.Fatal(err)
   }
   for _, dir := range dirs {
      _, err := os.Stat(dir.Name() + "/.git")
      if err == nil {
         cmd := exec.Command("git", "status")
         cmd.Dir = dir.Name()
         cmd.Stderr = os.Stderr
         cmd.Stdout = os.Stdout
         fmt.Printf("%v Dir %v %v\n", invert, reset, cmd.Dir)
         err := cmd.Run()
         if err != nil {
            log.Fatal(err)
         }
      }
   }
}
