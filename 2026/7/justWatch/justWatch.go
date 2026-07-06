package main

import (
   "41.neocities.org/justWatch"
   "bytes"
   "errors"
   "flag"
   "fmt"
   "log"
   "os"
   "path"
   "strings"
   "time"
)

func main() {
   log.SetFlags(log.Ltime)
   err := new(client).do()
   if err != nil {
      log.Fatal(err)
   }
}

type client struct {
   address string
   filters string
   sleep   time.Duration
}

func (c *client) do() error {
   flag.StringVar(&c.address, "a", "", "address")
   flag.DurationVar(&c.sleep, "s", 99*time.Millisecond, "sleep")
   flag.StringVar(&c.filters, "f", "BUY,CINEMA,FAST,RENT", "filters")
   flag.Parse()

   if c.address != "" {
      return c.do_address()
   }
   flag.Usage()
   return nil
}

func (c *client) do_address() error {
   url_path, err := justWatch.GetPath(c.address)
   if err != nil {
      return err
   }
   var content justWatch.Content
   err = content.Fetch(url_path)
   if err != nil {
      return err
   }
   var allEnrichedOffers []*justWatch.EnrichedOffer
   for _, tag := range content.HrefLangTags {
      locale, ok := justWatch.EnUs.Locale(&tag)
      if !ok {
         return errors.New("Locale")
      }
      log.Print(locale)
      offers, err := tag.Offers(locale)
      if err != nil {
         return err
      }
      for _, offer := range offers {
         allEnrichedOffers = append(allEnrichedOffers,
            &justWatch.EnrichedOffer{Locale: locale, Offer: &offer},
         )
      }
      time.Sleep(c.sleep)
   }
   enrichedOffers := justWatch.Deduplicate(allEnrichedOffers)
   enrichedOffers = justWatch.FilterOffers(
      enrichedOffers, strings.Split(c.filters, ",")...,
   )
   sortedUrls, groupedOffers := justWatch.GroupAndSortByUrl(enrichedOffers)
   data := &bytes.Buffer{}
   for i, address := range sortedUrls {
      if i >= 1 {
         data.WriteString("\n\n")
      }
      data.WriteString("## ")
      data.WriteString(address)
      for _, enriched := range groupedOffers[address] {
         data.WriteByte('\n')
         data.WriteString("\ncountry: ")
         data.WriteString(enriched.Locale.Country)
         data.WriteString("\nname: ")
         data.WriteString(enriched.Locale.CountryName)
         data.WriteString("\nmonetization: ")
         data.WriteString(enriched.Offer.MonetizationType)
         if enriched.Offer.ElementCount >= 1 {
            data.WriteString("\ncount: ")
            fmt.Fprint(data, enriched.Offer.ElementCount)
         }
      }
   }
   name := path.Base(url_path) + ".md"
   log.Println("WriteFile", name)
   return os.WriteFile(name, data.Bytes(), os.ModePerm)
}
