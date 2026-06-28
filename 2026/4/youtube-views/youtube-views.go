package main

import (
   "bytes"
   "encoding/json"
   "flag"
   "log"
   "net/http"
   "net/url"
   "strconv"
   "strings"
   "time"
)

const (
   web         = "WEB"
   web_version = "2.20231219.04.00"
)

const user_agent = "com.google.android.youtube/"

func format_integer(number int) string {
   number_string := strconv.Itoa(number)
   lengthOfString := len(number_string)
   if lengthOfString <= 3 {
      return number_string
   }
   digits := lengthOfString % 3
   if digits == 0 {
      digits = 3
   }
   var data strings.Builder
   data.WriteString(number_string[:digits])
   for i := digits; i < lengthOfString; i += 3 {
      data.WriteByte(',')
      data.WriteString(number_string[i : i+3])
   }
   return data.String()
}

func main() {
   http.DefaultTransport = &http.Transport{
      Proxy: func(req *http.Request) (*url.URL, error) {
         log.Println(req.Method, req.URL)
         return nil, nil
      },
   }
   log.SetFlags(log.Ltime)
   var tube InnerTube
   tube.Context.Client.ClientName = web
   flag.StringVar(&tube.VideoId, "v", "", "video ID")
   flag.Parse()
   if tube.VideoId != "" {
      err := tube.do()
      if err != nil {
         log.Fatal(err)
      }
   } else {
      flag.Usage()
   }
}

func views_per_year(views int, publish date) int {
   log.Println("publish", publish[0])
   years := time.Since(publish[0]).Hours() / 24 / 365
   return int(float64(views) / years)
}

// need `osVersion` this to get the correct:
// This video requires payment to watch
// instead of the invalid:
// This video can only be played on newer versions of Android or other
// supported devices.
type InnerTube struct {
   ContentCheckOk bool `json:"contentCheckOk,omitempty"`
   Context        struct {
      Client struct {
         AndroidSdkVersion int    `json:"androidSdkVersion"`
         ClientName        string `json:"clientName"`
         ClientVersion     string `json:"clientVersion"`
         OsVersion         string `json:"osVersion"`
      } `json:"client"`
   } `json:"context"`
   RacyCheckOk bool   `json:"racyCheckOk,omitempty"`
   VideoId     string `json:"videoId"`
}

func (i *InnerTube) Player() (*Player, error) {
   i.Context.Client.AndroidSdkVersion = 32
   i.Context.Client.ClientVersion = web_version
   i.Context.Client.OsVersion = "12"
   data, err := json.Marshal(i)
   if err != nil {
      return nil, err
   }
   req, err := http.NewRequest(
      "POST", "https://www.youtube.com/youtubei/v1/player",
      bytes.NewReader(data),
   )
   if err != nil {
      return nil, err
   }
   req.Header.Set("user-agent", user_agent+i.Context.Client.ClientVersion)
   resp, err := http.DefaultClient.Do(req)
   if err != nil {
      return nil, err
   }
   defer resp.Body.Close()
   play := &Player{}
   err = json.NewDecoder(resp.Body).Decode(play)
   if err != nil {
      return nil, err
   }
   return play, nil
}

func (i *InnerTube) do() error {
   play, err := i.Player()
   if err != nil {
      return err
   }
   views := views_per_year(
      play.VideoDetails.ViewCount,
      play.Microformat.PlayerMicroformatRenderer.PublishDate,
   )
   const limit = 10_000_000
   log.Println("video ID", play.VideoDetails.VideoId)
   log.Println("limit", format_integer(limit))
   log.Println("views", format_integer(views))
   return nil
}

type Player struct {
   Microformat struct {
      PlayerMicroformatRenderer struct {
         PublishDate date
      }
   }
   VideoDetails struct {
      VideoId   string
      ViewCount int `json:",string"`
   }
}

type date [1]time.Time

func (d *date) UnmarshalText(data []byte) error {
   var err error
   d[0], err = time.Parse(time.RFC3339, string(data))
   return err
}
