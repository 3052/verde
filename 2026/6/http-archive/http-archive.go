package main

import (
   "encoding/json"
   "flag"
   "fmt"
   "os"
   "strings"
)

type HAR struct {
   Log struct {
      Version string            `json:"version"`
      Creator interface{}       `json:"creator"`
      Pages   interface{}       `json:"pages,omitempty"`
      Entries []json.RawMessage `json:"entries"`
   } `json:"log"`
}

type PartialEntry struct {
   StartedDateTime string `json:"startedDateTime"`
   Request         struct {
      URL      string `json:"url"`
      PostData struct {
         MimeType string          `json:"mimeType"`
         Text     string          `json:"text"`
         Params   []NameValuePair `json:"params"`
      } `json:"postData"`
      Headers     []NameValuePair `json:"headers"`
      Cookies     []NameValuePair `json:"cookies"`
      QueryString []NameValuePair `json:"queryString"`
   } `json:"request"`
   Response struct {
      Content struct {
         Text string `json:"text"`
      } `json:"content"`
      Headers []NameValuePair `json:"headers"`
      Cookies []NameValuePair `json:"cookies"`
   } `json:"response"`
}

type NameValuePair struct {
   Name  string `json:"name"`
   Value string `json:"value"`
}

func main() {
   var inputFile, targetTime string
   var headerFlag, queryFlag, jsonFlag, formFlag, cookieFlag string

   flag.StringVar(&inputFile, "in", "", "Path to the input .har file (required)")
   flag.StringVar(&targetTime, "time", "", "startedDateTime value of the target request (required)")

   flag.StringVar(&headerFlag, "header", "", "Name of the Request Header to trace")
   flag.StringVar(&queryFlag, "query", "", "Name of the URL Query Parameter to trace")
   flag.StringVar(&jsonFlag, "json", "", "Name of the JSON body key to trace")
   flag.StringVar(&formFlag, "form", "", "Name of the Form data key to trace")
   flag.StringVar(&cookieFlag, "cookie", "", "Name of the Cookie to trace")

   flag.Parse()

   if inputFile == "" || targetTime == "" {
      fmt.Fprintln(os.Stderr, "Error: Missing required arguments (-in and -time).")
      flag.PrintDefaults()
      os.Exit(1)
   }

   // Validate that at most ONE trace flag is provided
   flagsUsed := 0
   var traceType, traceKey string

   if headerFlag != "" {
      flagsUsed++
      traceType = "header"
      traceKey = headerFlag
   }
   if queryFlag != "" {
      flagsUsed++
      traceType = "query"
      traceKey = queryFlag
   }
   if jsonFlag != "" {
      flagsUsed++
      traceType = "json"
      traceKey = jsonFlag
   }
   if formFlag != "" {
      flagsUsed++
      traceType = "form"
      traceKey = formFlag
   }
   if cookieFlag != "" {
      flagsUsed++
      traceType = "cookie"
      traceKey = cookieFlag
   }

   if flagsUsed > 1 {
      fmt.Fprintln(os.Stderr, "Fatal error: Only one trace flag (-header, -query, -json, -form, -cookie) can be used at a time.")
      os.Exit(1)
   }

   // Create Windows-safe output filename from time string
   safeTime := strings.ReplaceAll(targetTime, ":", "-")
   safeTime = strings.ReplaceAll(safeTime, "<", "_")
   safeTime = strings.ReplaceAll(safeTime, ">", "_")
   outputFile := safeTime + ".har"

   if err := processHAR(inputFile, targetTime, outputFile, traceType, traceKey); err != nil {
      fmt.Fprintf(os.Stderr, "Fatal error: %v\n", err)
      os.Exit(1)
   }
}

func processHAR(inputFile, targetTime, outputFile, traceType, traceKey string) error {
   data, err := os.ReadFile(inputFile)
   if err != nil {
      return fmt.Errorf("error reading file: %w", err)
   }

   var har HAR
   if err := json.Unmarshal(data, &har); err != nil {
      return fmt.Errorf("error parsing HAR JSON: %w", err)
   }

   entries := make([]PartialEntry, len(har.Log.Entries))
   targetIdx := -1

   // 1. Find the Target Request
   for i, rawEntry := range har.Log.Entries {
      if err := json.Unmarshal(rawEntry, &entries[i]); err != nil {
         continue
      }
      if entries[i].StartedDateTime == targetTime {
         targetIdx = i
         break
      }
   }

   if targetIdx == -1 {
      return fmt.Errorf("target request with the specified startedDateTime not found")
   }

   // 2. If NO config flags were provided, just output the chosen request and exit
   if traceType == "" {
      har.Log.Entries = []json.RawMessage{har.Log.Entries[targetIdx]}
      return writeOutput(har, outputFile)
   }

   // 3. Extract the configured value from the Target Request
   traceValue, found := extractValue(&entries[targetIdx], traceType, traceKey)
   if !found {
      return fmt.Errorf("the specified %s key '%s' was not found in the target request", traceType, traceKey)
   }

   // 4. Find the matching parent in previous responses
   var matchIndices []int
   for i := targetIdx - 1; i >= 0; i-- {
      if responseContainsValue(&entries[i], traceKey, traceValue) {
         matchIndices = append(matchIndices, i)
      }
   }

   // 5. Evaluate matches
   if len(matchIndices) == 0 {
      return fmt.Errorf("found 0 previous responses providing the key '%s' with value '%s'", traceKey, traceValue)
   }
   if len(matchIndices) > 1 {
      return fmt.Errorf("found %d previous responses providing the key '%s' with value '%s'. Expected exactly 1 match", len(matchIndices), traceKey, traceValue)
   }

   parentIdx := matchIndices[0]

   // 6. Write chronological output
   har.Log.Entries = []json.RawMessage{
      har.Log.Entries[parentIdx],
      har.Log.Entries[targetIdx],
   }

   return writeOutput(har, outputFile)
}

func extractValue(req *PartialEntry, traceType, traceKey string) (string, bool) {
   switch traceType {
   case "header":
      for _, h := range req.Request.Headers {
         if strings.EqualFold(h.Name, traceKey) {
            val := strings.TrimPrefix(h.Value, "Bearer ")
            return strings.TrimSpace(val), true
         }
      }
   case "query":
      for _, q := range req.Request.QueryString {
         if strings.EqualFold(q.Name, traceKey) {
            return strings.TrimSpace(q.Value), true
         }
      }
   case "cookie":
      for _, c := range req.Request.Cookies {
         if strings.EqualFold(c.Name, traceKey) {
            return strings.TrimSpace(c.Value), true
         }
      }
   case "form":
      for _, p := range req.Request.PostData.Params {
         if strings.EqualFold(p.Name, traceKey) {
            return strings.TrimSpace(p.Value), true
         }
      }
   case "json":
      if req.Request.PostData.Text != "" {
         var payload interface{}
         if err := json.Unmarshal([]byte(req.Request.PostData.Text), &payload); err == nil {
            var foundVal string
            var found bool
            findJSONKey(payload, traceKey, func(v string) {
               if !found { // Take the first matching key
                  foundVal = strings.TrimSpace(v)
                  found = true
               }
            })
            if found {
               return foundVal, true
            }
         }
      }
   }
   return "", false
}

func responseContainsValue(res *PartialEntry, traceKey, traceValue string) bool {
   // 1. Check Cookies
   for _, c := range res.Response.Cookies {
      if strings.EqualFold(c.Name, traceKey) {
         if c.Value == traceValue {
            return true
         }
      }
   }

   // 2. Check Headers
   for _, h := range res.Response.Headers {
      if strings.EqualFold(h.Name, traceKey) {
         if h.Value == traceValue {
            return true
         }
      }
   }

   // 3. Check JSON Body
   if res.Response.Content.Text != "" {
      var payload interface{}
      if err := json.Unmarshal([]byte(res.Response.Content.Text), &payload); err == nil {
         found := false
         findJSONKey(payload, traceKey, func(extracted string) {
            if extracted == traceValue {
               found = true
            }
         })
         if found {
            return true
         }
      }
   }

   return false
}

func findJSONKey(data interface{}, targetKey string, processValue func(string)) {
   switch val := data.(type) {
   case map[string]interface{}:
      for k, v := range val {
         if strings.EqualFold(k, targetKey) {
            switch castV := v.(type) {
            case string:
               processValue(castV)
            case float64:
               processValue(fmt.Sprintf("%v", castV))
            case bool:
               processValue(fmt.Sprintf("%t", castV))
            }
         }
         findJSONKey(v, targetKey, processValue)
      }
   case []interface{}:
      for _, item := range val {
         findJSONKey(item, targetKey, processValue)
      }
   }
}

func writeOutput(har HAR, outputFile string) error {
   outData, err := json.MarshalIndent(har, "", "  ")
   if err != nil {
      return fmt.Errorf("error marshalling output: %w", err)
   }

   if err := os.WriteFile(outputFile, outData, 0644); err != nil {
      return fmt.Errorf("error writing to output file: %w", err)
   }

   fmt.Printf("Success! Output saved to %s\n", outputFile)
   return nil
}
