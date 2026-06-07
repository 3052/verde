package main

import (
   "encoding/json"
   "fmt"
   "strings"
)

// Accepts *TraceConfig
func extractExplicitValues(req *PartialEntry, cfg *TraceConfig) []string {
   var extracted []string

   addValue := func(val string) {
      val = strings.TrimSpace(val)
      val = strings.TrimPrefix(val, "Bearer ")
      if val != "" {
         extracted = append(extracted, val)
      }
   }

   for _, targetHeader := range cfg.Headers {
      for _, h := range req.Request.Headers {
         if strings.EqualFold(h.Name, targetHeader) {
            addValue(h.Value)
         }
      }
   }

   for _, targetQuery := range cfg.Queries {
      for _, q := range req.Request.QueryString {
         if strings.EqualFold(q.Name, targetQuery) {
            addValue(q.Value)
         }
      }
   }

   if len(cfg.JSONKeys) > 0 && req.Request.PostData.Text != "" {
      var payload interface{}
      if err := json.Unmarshal([]byte(req.Request.PostData.Text), &payload); err == nil {
         for _, targetKey := range cfg.JSONKeys {
            findJSONKey(payload, targetKey, addValue)
         }
      }
   }

   return extracted
}

func findJSONKey(data interface{}, targetKey string, addValue func(string)) {
   switch val := data.(type) {
   case map[string]interface{}:
      for k, v := range val {
         if strings.EqualFold(k, targetKey) {
            switch castV := v.(type) {
            case string:
               addValue(castV)
            case float64:
               addValue(fmt.Sprintf("%v", castV))
            case bool:
               addValue(fmt.Sprintf("%t", castV))
            }
         }
         findJSONKey(v, targetKey, addValue)
      }
   case []interface{}:
      for _, item := range val {
         findJSONKey(item, targetKey, addValue)
      }
   }
}

func responseContainsValue(res *PartialEntry, value string) bool {
   for _, h := range res.Response.Headers {
      if strings.Contains(h.Value, value) {
         return true
      }
   }

   return strings.Contains(res.Response.Content.Text, value)
}
