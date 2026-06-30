package main

import (
   "encoding/xml"
   "flag"
   "fmt"
   "iter"
   "log"
   "os"
   "strings"
)

func do(name string) error {
   data, err := os.ReadFile(name)
   if err != nil {
      return err
   }
   var manifestVar manifest
   err = xml.Unmarshal(data, &manifestVar)
   if err != nil {
      return err
   }
   for intent := range manifestVar.intent_filter() {
      fmt.Print(&intent, "\n\n")
   }
   return nil
}

func main() {
   name := flag.String("n", "", "name")
   flag.Parse()
   if *name != "" {
      err := do(*name)
      if err != nil {
         log.Fatal(err)
      }
   } else {
      flag.Usage()
   }
}

func (i *intent_filter) String() string {
   var b strings.Builder
   b.WriteString("action.name = ")
   b.WriteString(i.Action.Name)
   for _, category := range i.Category {
      b.WriteString("\ncategory.name = ")
      b.WriteString(category.Name)
   }
   for _, data := range i.Data {
      if data.Host != "" {
         b.WriteString("\ndata.host = ")
         b.WriteString(data.Host)
      }
      if data.PathPattern != "" {
         b.WriteString("\ndata.pathPattern = ")
         b.WriteString(data.PathPattern)
      }
      if data.PathPrefix != "" {
         b.WriteString("\ndata.pathPrefix = ")
         b.WriteString(data.PathPrefix)
      }
      if data.Scheme != "" {
         b.WriteString("\ndata.scheme = ")
         b.WriteString(data.Scheme)
      }
   }
   return b.String()
}

type manifest struct {
   Application struct {
      Activity []struct {
         IntentFilter []intent_filter `xml:"intent-filter"`
      } `xml:"activity"`
   } `xml:"application"`
}

type intent_filter struct {
   Action struct {
      Name string `xml:"name,attr"`
   } `xml:"action"`
   Category []struct {
      Name string `xml:"name,attr"`
   } `xml:"category"`
   Data []struct {
      Scheme      string `xml:"scheme,attr"`
      Host        string `xml:"host,attr"`
      PathPattern string `xml:"pathPattern,attr"`
      PathPrefix  string `xml:"pathPrefix,attr"`
   } `xml:"data"`
}

func (m manifest) intent_filter() iter.Seq[intent_filter] {
   return func(yield func(intent_filter) bool) {
      for _, activity := range m.Application.Activity {
         for _, intent := range activity.IntentFilter {
            if intent.Action.Name == "android.intent.action.VIEW" {
               if len(intent.Data) >= 1 {
                  if !yield(intent) {
                     return
                  }
               }
            }
         }
      }
   }
}
