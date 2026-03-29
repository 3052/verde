package main

import (
   "testing"
)

// Global variable to prevent the compiler from optimizing the function calls away
var result string

func BenchmarkStringAppend(b *testing.B) {
   rec := Record{
      FirstName: "Jane",
      LastName:  "Doe",
      Age:       28,
      ID:        1024,
   }

   b.ResetTimer()
   for i := 0; i < b.N; i++ {
      result = rec.StringAppend()
   }
}

func BenchmarkStringFprint(b *testing.B) {
   rec := Record{
      FirstName: "Jane",
      LastName:  "Doe",
      Age:       28,
      ID:        1024,
   }

   b.ResetTimer()
   for i := 0; i < b.N; i++ {
      result = rec.StringFprint()
   }
}
