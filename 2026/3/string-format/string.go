package main

import (
   "fmt"
   "strings"
)

// Record has two strings and two ints.
type Record struct {
   FirstName string
   LastName  string
   Age       int
   ID        int
}

// Method 1: Using fmt.Append once per field.
// We chain the byte slice `b` through multiple Append calls.
func (r Record) StringAppend() string {
   var b []byte
   b = fmt.Append(b, "FirstName: ", r.FirstName, " ")
   b = fmt.Append(b, "LastName: ", r.LastName, " ")
   b = fmt.Append(b, "Age: ", r.Age, " ")
   b = fmt.Append(b, "ID: ", r.ID)
   return string(b) // Convert the final byte slice to a string
}

// Method 2: Using fmt.Fprint and strings.Builder once per field.
func (r Record) StringFprint() string {
   var sb strings.Builder
   fmt.Fprint(&sb, "FirstName: ", r.FirstName, " ")
   fmt.Fprint(&sb, "LastName: ", r.LastName, " ")
   fmt.Fprint(&sb, "Age: ", r.Age, " ")
   fmt.Fprint(&sb, "ID: ", r.ID)
   return sb.String()
}

func main() {
   rec := Record{
      FirstName: "Jane",
      LastName:  "Doe",
      Age:       28,
      ID:        1024,
   }

   fmt.Println("Append Method:", rec.StringAppend())
   fmt.Println("Fprint Method:", rec.StringFprint())
}
