package main

import (
   "encoding/json"
   "fmt"
   "strings"
)

func extractExplicitValues(req *PartialEntry, cfg *TraceConfig) []string {
   var extracted []string
   seen := make(map[string]bool)

   addValue := func(val string) {
      val = strings.TrimSpace(val)
      val = strings.TrimPrefix(val, "Bearer ")
      if val != "" && !seen[val] {
         seen[val] = true
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

   // Extract specified Cookies
   for _, targetCookie := range cfg.Cookies {
      for _, c := range req.Request.Cookies {
         if strings.EqualFold(c.Name, targetCookie) {
            addValue(c.Value)
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

   for _, targetKey := range cfg.FormKeys {
      for _, p := range req.Request.PostData.Params {
         if strings.EqualFold(p.Name, targetKey) {
            addValue(p.Value)
         }
      }
   }

   return extracted
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

func responseContainsValue(res *PartialEntry, cfg *TraceConfig, value string) bool {

   // 1. Check explicitly configured Response Cookies strictly
   for _, targetCookie := range cfg.Cookies {
      for _, c := range res.Response.Cookies {
         if strings.EqualFold(c.Name, targetCookie) {
            if c.Value == value { // EXACT MATCH
               return true
            }
         }
      }
   }

   // 2. Check explicitly configured Response Headers strictly
   for _, targetHeader := range cfg.Headers {
      for _, h := range res.Response.Headers {
         if strings.EqualFold(h.Name, targetHeader) {
            if h.Value == value { // EXACT MATCH
               return true
            }
         }
      }
   }

   // 3. Check explicitly configured JSON keys in the Response Body strictly
   if len(cfg.JSONKeys) > 0 && res.Response.Content.Text != "" {
      var payload interface{}
      if err := json.Unmarshal([]byte(res.Response.Content.Text), &payload); err == nil {
         found := false

         checkMatch := func(extracted string) {
            if extracted == value { // EXACT MATCH
               found = true
            }
         }

         for _, targetKey := range cfg.JSONKeys {
            findJSONKey(payload, targetKey, checkMatch)
         }

         if found {
            return true
         }
      }
   }

   return false
}
