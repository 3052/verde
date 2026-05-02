package main

import (
   "41.neocities.org/verde/nordVpn"
   "encoding/json"
   "errors"
   "flag"
   "fmt"
   "io"
   "log"
   "os"
   "os/exec"
   "path/filepath"
   "time"
)

func main() {
   log.SetFlags(log.Ltime)
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

func (c *client) do_country_code() error {
   data, err := read_file(c.cache)
   if err != nil {
      return err
   }
   servers, err := nordVpn.ReadServers(data)
   if err != nil {
      return err
   }
   data, err = exec.Command("credential", "-j=api.nordvpn.com").Output()
   if err != nil {
      return err
   }
   var credential []struct {
      Username string
      Password string
   }
   err = json.Unmarshal(data, &credential)
   if err != nil {
      return err
   }
   for _, server := range servers {
      if server.ProxySsl() {
         if server.Country(c.country_code) {
            fmt.Println(
               nordVpn.FormatProxy(
                  credential[0].Username, credential[0].Password,
                  server.Hostname,
               ),
            )
         }
      }
   }
   return nil
}

const duration = 24 * time.Hour

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
   if time.Since(info.ModTime()) >= duration {
      return nil, errors.New(duration.String())
   }
   return io.ReadAll(file)
}
