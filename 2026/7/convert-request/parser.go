package main

import (
   "bufio"
   "bytes"
   "encoding/json"
   "io"
   "net/url"
   "os"
   "strconv"
   "strings"
)

type Cookie struct {
   Name  string
   Value string
}

type QueryParam struct {
   Key   string
   Value string
}

type RequestData struct {
   Method    string
   URLScheme string
   URLHost   string
   URLPath   string
   URLQuery  []QueryParam
   Headers   map[string]string
   Cookies   []Cookie
   Body      string
   HasBody   bool
}

func parseRawRequest(filename string, indentBody bool) (*RequestData, error) {
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
      return nil, err
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
   _, err = io.Copy(&bodyBytes, reader)
   if err != nil && err != io.EOF {
      return nil, err
   }

   if bodyBytes.Len() > 0 {
      reqData.HasBody = true
      bodyStr := bodyBytes.String()

      if indentBody {
         var indented bytes.Buffer
         if err := json.Indent(&indented, bodyBytes.Bytes(), "", "\t"); err == nil {
            bodyStr = indented.String()
         }
      }

      reqData.Body = "`" + strings.ReplaceAll(bodyStr, "`", "`+\"`\"+`") + "`"
   }

   return reqData, nil
}
