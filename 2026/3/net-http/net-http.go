package main

import (
   "bufio"
   "bytes"
   "embed"
   "flag"
   "fmt"
   "io"
   "log"
   "net/http"
   "net/textproto"
   "net/url"
   "os"
   "strings"
   "text/template"
)

func (c *client) do() error {
   flag.BoolVar(&c.form, "f", false, "form")
   flag.BoolVar(&c.golang, "g", false, "request as Go code")
   flag.BoolVar(&c.https, "s", false, "HTTPS")
   flag.StringVar(&c.in.name, "i", "", "in file")
   flag.StringVar(&c.out.name, "o", "", "output file")
   flag.Parse()
   if c.in.name != "" {
      var err error
      c.out.file, err = new_file(c.out.name)
      if err != nil {
         return err
      }
      defer c.out.file.Close()
      c.in.file, err = os.Open(c.in.name)
      if err != nil {
         return err
      }
      defer c.in.file.Close()
      req, err := read_request(bufio.NewReader(c.in.file))
      if err != nil {
         return err
      }
      if req.URL.Scheme == "" {
         if c.https {
            req.URL.Scheme = "https"
         } else {
            req.URL.Scheme = "http"
         }
      }
      if c.golang {
         return c.write_go(req)
      }
      return c.write(req)
   }
   flag.Usage()
   return nil
}

func main() {
   err := new(client).do()
   if err != nil {
      log.Fatal(err)
   }
}

func (c *client) write(req *http.Request) error {
   resp, err := http.DefaultClient.Do(req)
   if err != nil {
      return err
   }
   if c.out.name != "" {
      // 1. body to file
      _, err = c.out.file.ReadFrom(resp.Body)
      if err != nil {
         return err
      }
      resp.ContentLength = 0
   }
   // 2. head to stdout
   return resp.Write(os.Stdout)
}

func (c *client) write_go(req *http.Request) error {
   var value request
   value.Method = req.Method
   value.URL = req.URL
   value.Header = req.Header
   if req.Body != nil {
      var data strings.Builder
      _, err := io.Copy(&data, req.Body)
      if err != nil {
         return err
      }
      if c.form {
         form, err := url.ParseQuery(data.String())
         if err != nil {
            return err
         }
         value.RawBody = fmt.Sprintf("\n%#v.Encode(),\n", form)
      } else {
         value.RawBody = fmt.Sprintf("%#q", data.String())
      }
      value.Body = "io.NopCloser(strings.NewReader(data))"
   } else {
      value.RawBody = `""`
      value.Body = "nil"
   }
   temp, err := template.ParseFS(content, ".net-http.go")
   if err != nil {
      return err
   }
   return temp.Execute(c.out.file, value)
}

type client struct {
   golang bool
   https  bool
   form   bool
   in     struct {
      name string
      file *os.File
   }
   out struct {
      name string
      file *os.File
   }
}

type request struct {
   Method  string
   URL     *url.URL
   Header  http.Header
   Body    string
   RawBody string
}

func new_file(name string) (*os.File, error) {
   if name != "" {
      return os.Create(name)
   }
   return os.Stdout, nil
}
//go:embed .net-http.go
var content embed.FS

// this is needed because http.ReadRequest is trash
func read_request(r *bufio.Reader) (*http.Request, error) {
   var req http.Request
   text := textproto.NewReader(r)
   // .Method
   raw_method_path, err := text.ReadLine()
   if err != nil {
      return nil, err
   }
   method_path := strings.Fields(raw_method_path)
   req.Method = method_path[0]
   // .URL
   ref, err := url.ParseRequestURI(method_path[1])
   if err != nil {
      return nil, err
   }
   req.URL = ref
   // .URL.Host
   head, err := text.ReadMIMEHeader()
   if err != nil {
      return nil, err
   }
   if req.URL.Host == "" {
      req.URL.Host = head.Get("Host")
   }
   // .Header
   req.Header = http.Header(head)
   // .Body
   data := &bytes.Buffer{}
   length, err := text.R.WriteTo(data)
   if err != nil {
      return nil, err
   }
   if length >= 1 {
      req.Body = io.NopCloser(data)
   }
   // .ContentLength
   req.ContentLength = length
   return &req, nil
}
