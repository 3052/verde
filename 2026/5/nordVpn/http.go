package nordVpn

import (
   "log"
   "net/http"
   "net/url"
)

func Get(targetUrl *url.URL, headers map[string]string) (*http.Response, error) {
   reqHeader := make(http.Header)
   for key, value := range headers {
      reqHeader.Set(key, value)
   }
   req := &http.Request{
      Method: http.MethodGet,
      URL:    targetUrl,
      Header: reqHeader,
   }

   log.Println(req.Method, req.URL)
   return http.DefaultClient.Do(req)
}
