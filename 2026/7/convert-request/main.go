// main.go
package main

import (
   "bytes"
   "flag"
   "go/format"
   "log"
   "os"
   "text/template"
)

const goCodeTemplate = `package main

import (
{{if or .HasBody .HasFormBody}}   "bytes"
{{end}}   "net/http"
   "net/url"
   "os"
)

func main() {
   client := &http.Client{}

   reqURL := &url.URL{
      Scheme: {{.URLScheme}},
      Host:   {{.URLHost}},
      Path:   {{.URLPath}},
   }
{{if .URLQuery}}
   q := url.Values{}
{{range .URLQuery}}   q.Add({{.Key}}, {{.Value}})
{{end}}   reqURL.RawQuery = q.Encode()
{{end}}

{{if .HasFormBody}}
   form := url.Values{}
{{range .FormParams}}   form.Add({{.Key}}, {{.Value}})
{{end}}   req, err := http.NewRequest("{{.Method}}", reqURL.String(), bytes.NewBufferString(form.Encode()))
{{else if .HasBody}}
   bodyData := []byte({{.Body}})
   req, err := http.NewRequest("{{.Method}}", reqURL.String(), bytes.NewBuffer(bodyData))
{{else}}
   req, err := http.NewRequest("{{.Method}}", reqURL.String(), nil)
{{end}}
   if err != nil {
      panic(err)
   }

{{range $key, $value := .Headers}}   req.Header.Add("{{$key}}", {{$value}})
{{end}}
{{range .Cookies}}   req.AddCookie(&http.Cookie{Name: {{.Name}}, Value: {{.Value}}})
{{end}}
   resp, err := client.Do(req)
   if err != nil {
      panic(err)
   }
   if err := resp.Write(os.Stdout); err != nil {
      panic(err)
   }
}

`

func generateGoFile(data *RequestData, outputPath string) error {
   tmpl, err := template.New("").Parse(goCodeTemplate)
   if err != nil {
      return err
   }

   var buf bytes.Buffer
   if err = tmpl.Execute(&buf, data); err != nil {
      return err
   }

   formattedCode, err := format.Source(buf.Bytes())
   if err != nil {
      formattedCode = buf.Bytes()
   }

   return os.WriteFile(outputPath, formattedCode, 0644)
}

func main() {
   inputFile := flag.String("in", "", "Input HTTP request text file (required)")
   indentJSON := flag.Bool("indent", false, "Pretty-print/indent the JSON body")
   forceForm := flag.Bool("form", false, "Force treating the body as a form-encoded query string (key=value&...)")

   flag.Parse()

   if *inputFile == "" {
      flag.Usage()
      os.Exit(1)
   }

   reqData, err := parseRawRequest(*inputFile, *indentJSON, *forceForm)
   if err != nil {
      log.Fatalf("Error parsing request: %v", err)
   }

   outputFile := *inputFile + ".go"
   if err = generateGoFile(reqData, outputFile); err != nil {
      log.Fatalf("Error generating file: %v", err)
   }

   log.Printf("Success: Generated Go code written to %s\n", outputFile)
}
