package main

import (
   "flag"
   "fmt"
   "net/http"
)

func main() {
   port := flag.String("p", ":8080", "port")
   strip := flag.String("s", "", "strip")
   flag.Parse()
   fmt.Printf("localhost%v%v\n", *port, *strip)
   http.ListenAndServe(
      *port, http.StripPrefix(
         *strip, http.FileServer(http.Dir("")),
      ),
   )
}
