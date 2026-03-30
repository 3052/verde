package main

import (
   "encoding/json"
   "flag"
   "fmt"
   "log"
   "os"
   "slices"
   "strings"
   "time"
)

const credential_json = `D:\backblaze\largest\credential.json`

type userinfo map[string]string

func (u userinfo) String() string {
   keys := make([]string, 0, len(u))
   for key := range u {
      keys = append(keys, key)
   }
   slices.Sort(keys)
   var data strings.Builder
   for i, key := range keys {
      if i >= 1 {
         data.WriteByte('\n')
      }
      data.WriteString(key)
      data.WriteString(" = ")
      data.WriteString(u[key])
   }
   return data.String()
}

func get_users() ([]userinfo, error) {
   data, err := os.ReadFile(credential_json)
   if err != nil {
      return nil, err
   }
   var users []userinfo
   err = json.Unmarshal(data, &users)
   if err != nil {
      return nil, err
   }
   passwords := map[string]bool{}
   for _, user := range users {
      trial := user["trial"]
      password := user["password"]
      if trial == "true" {
         trial2, ok := passwords[password]
         if ok {
            if !trial2 {
               return nil, fmt.Errorf("password conflict: trial user cannot use a non-trial password for user %v", user)
            }
         } else {
            passwords[password] = true
         }
      } else if trial == "false" {
         // This block is changed to reflect your logic.
         // We only check for duplicate passwords if the password is not empty.
         if password != "" {
            _, ok := passwords[password]
            if ok {
               return nil, fmt.Errorf("duplicate non-trial password for user: %v", user)
            } else {
               passwords[password] = false
            }
         }
         // If password is "", we now do nothing, allowing multiple non-trial users
         // to have an empty password.
      } else if password != "" {
         return nil, fmt.Errorf("user has a password but trial status is not 'true' or 'false': %v", user)
      }
   }
   year_ago := time.Now().AddDate(-1, 0, 0).String()
   for _, user := range users {
      if user["date"] < year_ago {
         return nil, fmt.Errorf("user account is older than one year: %v", user)
      }
   }
   return users, nil
}

func main() {
   key := flag.String("k", "password", "key")
   host := flag.String("h", "", "host")
   contains := flag.String("c", "", "contains")
   flag.Parse()
   if *key == "password" {
      if *contains == "" {
         if *host == "" {
            flag.Usage()
            return
         }
      }
   }
   users, err := get_users()
   if err != nil {
      log.Fatal(err)
   }
   var line bool
   for _, user2 := range users {
      if *contains != "" {
         if strings.Contains(user2.String(), *contains) {
            if line {
               fmt.Println()
            } else {
               line = true
            }
            fmt.Println(user2)
         }
      } else {
         if user2[*key] == "" {
            continue
         }
         if *host != "" {
            if user2["host"] != *host {
               continue
            }
         }
         fmt.Print(user2[*key])
         return
      }
   }
}
