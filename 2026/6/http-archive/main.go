package main

import (
   "flag"
   "fmt"
   "os"
)

func main() {
   inputFile := flag.String("in", "", "Path to the input .har file (required)")
   targetTime := flag.String("time", "", "startedDateTime value of the target request (required)")
   outputFile := flag.String("out", "", "Path to the output .har file (required)")

   flag.Parse()

   if *inputFile == "" || *targetTime == "" || *outputFile == "" {
      fmt.Println("Error: Missing required arguments.")
      fmt.Println("Usage:")
      flag.PrintDefaults()
      os.Exit(1)
   }

   if err := processHAR(*inputFile, *targetTime, *outputFile); err != nil {
      fmt.Fprintf(os.Stderr, "Fatal error: %v\n", err)
      os.Exit(1)
   }
}
