package main

import (
   "41.neocities.org/verde/nordVpn"
   "errors"
   "flag"
   "fmt"
   "io"
   "log"
   "net/http"
   "net/url"
   "os"
   "os/exec"
   "path/filepath"
   "strings"
   "time"
)

func main() {
   log.SetFlags(log.Ltime)
   http.DefaultTransport = &http.Transport{
      Proxy: func(req *http.Request) (*url.URL, error) {
         if req.Method == "" {
            req.Method = "GET"
         }
         log.Println(req.Method, req.URL)
         return nil, nil
      },
   }
   err := new(client).do()
   if err != nil {
      log.Fatal(err)
   }
}

func (c *client) do() error {
   var err error
   c.cache, err = os.UserCacheDir()
   if err != nil {
      return err
   }
   c.cache = filepath.Join(c.cache, "nordVpn/nordVpn.json")
   // 1
   flag.BoolVar(&c.write, "w", false, "write")
   // 2
   flag.StringVar(&c.country_code, "c", "", "country code")
   flag.Parse()
   if c.write {
      return c.do_write()
   }
   if c.country_code != "" {
      return c.do_country_code()
   }
   flag.Usage()
   return nil
}

func write_file(name string, data []byte) error {
   log.Println("WriteFile", name)
   return os.WriteFile(name, data, os.ModePerm)
}

func read_file(name string) ([]byte, error) {
   file, err := os.Open(name)
   if err != nil {
      return nil, err
   }
   defer file.Close()
   info, err := file.Stat()
   if err != nil {
      return nil, err
   }
   if time.Since(info.ModTime()) >= 24*time.Hour {
      return nil, errors.New("ModTime")
   }
   return io.ReadAll(file)
}

func (c *client) do_write() error {
   data, err := nordVpn.WriteServers(0)
   if err != nil {
      return err
   }
   return write_file(c.cache, data)
}

type client struct {
   cache string
   // 1
   write bool
   // 2
   country_code string
}

func output(name string, arg ...string) (string, error) {
   var data strings.Builder
   command := exec.Command(name, arg...)
   command.Stdout = &data
   log.Println("Run", command.Args)
   err := command.Run()
   if err != nil {
      return "", err
   }
   return data.String(), nil
}

func (c *client) do_country_code() error {
   data, err := read_file(c.cache)
   if err != nil {
      return err
   }
   servers, err := nordVpn.ReadServers(data)
   if err != nil {
      return err
   }
   username, err := output("credential", "-h=api.nordvpn.com", "-k=username")
   if err != nil {
      return err
   }
   password, err := output("credential", "-h=api.nordvpn.com")
   if err != nil {
      return err
   }
   for _, server := range servers {
      if server.ProxySsl() {
         if server.Country(c.country_code) {
            fmt.Println(
               nordVpn.FormatProxy(username, password, server.Hostname),
            )
         }
      }
   }
   return nil
}
