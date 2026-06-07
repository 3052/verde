package main

import (
   "encoding/json"
   "fmt"
   "net/url"
   "strings"
)

func extractPotentialTokens(req *PartialEntry) []string {
   tokenMap := make(map[string]bool)

   addToken := func(s string) {
      if len(s) >= 8 {
         tokenMap[s] = true
      }
   }

   for _, h := range req.Request.Headers {
      val := strings.TrimPrefix(h.Value, "Bearer ")
      addToken(val)
   }

   for _, c := range req.Request.Cookies {
      addToken(c.Value)
   }
   for _, q := range req.Request.QueryString {
      addToken(q.Value)
   }

   for _, p := range req.Request.PostData.Params {
      addToken(p.Value)
   }

   text := req.Request.PostData.Text
   if text != "" {
      var jsonData interface{}
      if err := json.Unmarshal([]byte(text), &jsonData); err == nil {
         extractJSONValues(jsonData, addToken)
      } else if strings.Contains(text, "=") {
         if parsedForm, err := url.ParseQuery(text); err == nil {
            for _, vals := range parsedForm {
               for _, v := range vals {
                  addToken(v)
               }
            }
         }
      }
   }

   if parsedURL, err := url.Parse(req.Request.URL); err == nil {
      segments := strings.Split(parsedURL.Path, "/")
      for _, segment := range segments {
         addToken(segment)
      }
   }

   var tokens []string
   for k := range tokenMap {
      tokens = append(tokens, k)
   }
   return tokens
}

func extractJSONValues(v interface{}, addToken func(string)) {
   switch val := v.(type) {
   case map[string]interface{}:
      for _, child := range val {
         extractJSONValues(child, addToken)
      }
   case []interface{}:
      for _, child := range val {
         extractJSONValues(child, addToken)
      }
   case string:
      addToken(val)
   case float64:
      addToken(fmt.Sprintf("%v", val))
   }
}

func responseContainsToken(res *PartialEntry, token string) bool {
   if strings.Contains(res.Response.Content.Text, token) {
      return true
   }
   for _, h := range res.Response.Headers {
      if strings.Contains(h.Value, token) {
         return true
      }
   }
   for _, c := range res.Response.Cookies {
      if strings.Contains(c.Value, token) {
         return true
      }
   }
   return false
}
