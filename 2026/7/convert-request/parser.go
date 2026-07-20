package main

import (
   "bufio"
   "bytes"
   "encoding/json"
   "fmt"
   "io"
   "net/url"
   "os"
   "strconv"
   "strings"
)

// containsControl reports whether s contains any byte that is unsafe to
// embed verbatim in a Go raw string literal. Newlines and tabs are allowed
// (they render fine inside backticks); everything else below 0x20, plus
// 0x7f (DEL), is treated as unsafe.
func containsControl(s string) bool {
   for i := 0; i < len(s); i++ {
      b := s[i]
      if b < 0x20 && b != '\n' && b != '\t' {
         return true
      }
      if b == 0x7f {
         return true
      }
   }
   return false
}

// goStringLiteral returns a Go string literal for s.
// It prefers a raw string literal (backticks) for readability when s
// contains no backticks and no problematic control characters; otherwise
// it falls back to strconv.Quote, which escapes everything safely.
func goStringLiteral(s string) string {
   if !strings.ContainsRune(s, '`') && !containsControl(s) {
      return "`" + s + "`"
   }
   return strconv.Quote(s)
}

type Cookie struct {
   Name  string
   Value string
}

type QueryParam struct {
   Key   string
   Value string
}

type RequestData struct {
   Method      string
   URLScheme   string
   URLHost     string
   URLPath     string
   URLQuery    []QueryParam
   Headers     map[string]string
   Cookies     []Cookie
   Body        string
   HasBody     bool
   HasFormBody bool
   FormParams  []QueryParam
}

func parseRawRequest(filename string, indentBody, forceForm bool) (*RequestData, error) {
   file, err := os.Open(filename)
   if err != nil {
      return nil, err
   }
   defer file.Close()

   reader := bufio.NewReader(file)
   reqData := &RequestData{
      Headers: make(map[string]string),
   }

   firstLine, err := reader.ReadString('\n')
   if err != nil {
      return nil, err
   }

   parts := strings.Split(strings.TrimSpace(firstLine), " ")
   if len(parts) < 3 {
      return nil, fmt.Errorf("invalid request line: %q", firstLine)
   }

   reqData.Method = parts[0]
   path := parts[1]

   host := ""
   for {
      line, err := reader.ReadString('\n')
      if err != nil && err != io.EOF {
         return nil, err
      }

      line = strings.TrimSpace(line)
      if line == "" {
         break
      }

      headerParts := strings.SplitN(line, ":", 2)
      if len(headerParts) == 2 {
         key := strings.TrimSpace(headerParts[0])
         value := strings.TrimSpace(headerParts[1])

         if strings.ToLower(key) == "cookie" {
            for _, c := range strings.Split(value, ";") {
               c = strings.TrimSpace(c)
               if c == "" {
                  continue
               }
               cParts := strings.SplitN(c, "=", 2)
               name := strings.TrimSpace(cParts[0])
               val := ""
               if len(cParts) == 2 {
                  val = strings.TrimSpace(cParts[1])
                  if strings.HasPrefix(val, `"`) && strings.HasSuffix(val, `"`) && len(val) >= 2 {
                     val = val[1 : len(val)-1]
                  }
               }
               reqData.Cookies = append(reqData.Cookies, Cookie{
                  Name:  strconv.Quote(name),
                  Value: strconv.Quote(val),
               })
            }
         } else {
            reqData.Headers[key] = strconv.Quote(value)
         }

         if strings.ToLower(key) == "host" {
            host = value
         }
      }

      if err == io.EOF {
         break
      }
   }

   var rawURL string
   if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
      rawURL = path
   } else {
      rawURL = "https://" + host + path
   }

   parsedURL, err := url.Parse(rawURL)
   if err != nil {
      return nil, err
   }

   reqData.URLScheme = strconv.Quote(parsedURL.Scheme)
   reqData.URLHost = strconv.Quote(parsedURL.Host)
   reqData.URLPath = strconv.Quote(parsedURL.Path)

   for k, vals := range parsedURL.Query() {
      for _, v := range vals {
         reqData.URLQuery = append(reqData.URLQuery, QueryParam{
            Key:   strconv.Quote(k),
            Value: strconv.Quote(v),
         })
      }
   }

   var bodyBytes bytes.Buffer
   if _, err = io.Copy(&bodyBytes, reader); err != nil && err != io.EOF {
      return nil, err
   }

   if bodyBytes.Len() > 0 {
      if forceForm {
         parsedForm, perr := url.ParseQuery(bodyBytes.String())
         if perr != nil {
            return nil, fmt.Errorf("failed to parse body as form-encoded: %w", perr)
         }
         reqData.HasFormBody = true
         for k, vals := range parsedForm {
            for _, v := range vals {
               reqData.FormParams = append(reqData.FormParams, QueryParam{
                  Key:   strconv.Quote(k),
                  Value: strconv.Quote(v),
               })
            }
         }
         if _, ok := reqData.Headers["Content-Type"]; !ok {
            reqData.Headers["Content-Type"] = strconv.Quote("application/x-www-form-urlencoded")
         }
         return reqData, nil
      }

      reqData.HasBody = true
      bodyStr := bodyBytes.String()
      if indentBody {
         var indented bytes.Buffer
         if err := json.Indent(&indented, bodyBytes.Bytes(), "", "\t"); err == nil {
            bodyStr = indented.String()
         }
      }
      reqData.Body = goStringLiteral(bodyStr)
   }

   return reqData, nil
}
