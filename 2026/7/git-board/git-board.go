package main

import (
   "fmt"
   "log"
   "os"
   "os/exec"
   "strings"
   "text/template"
   "time"
)

const (
   fail = "\x1b[30;101m Fail \x1b[m"
   pass = "\x1b[30;102m Pass \x1b[m"
)

const format = "{{ .AddStatus }} additions\ttarget:{{ .Target }}\tactual:{{ .Add }}\n" +
   "{{ .DeleteStatus }} deletions\ttarget:{{ .Target }}\tactual:{{ .Delete }}\n" +
   "{{ .ChangeStatus }} changed files\ttarget:{{ .Target}}\tactual:{{ .Change }}\n" +
   "{{ .DateStatus }} last commit\ttarget:{{ .Then }}\tactual:{{ .Now }}\n"

func main() {
   // Updated function call
   board, err := GenerateGitBoard()
   if err != nil {
      log.Fatal(err)
   }
   template_data, err := new(template.Template).Parse(format)
   if err != nil {
      log.Fatal(err)
   }
   if err := template_data.Execute(os.Stdout, board); err != nil {
      log.Fatal(err)
   }
}

func run(name string, args ...string) (string, error) {
   var data strings.Builder
   command := exec.Command(name, args...)
   command.Stdout = &data
   fmt.Println(command.Args)
   err := command.Run()
   if err != nil {
      return "", err
   }
   return data.String(), nil
}

type GitBoard struct {
   Add          int
   AddStatus    string
   Delete       int
   DeleteStatus string
   Change       int
   ChangeStatus string
   Target       int
   Then         string
   Now          string
   DateStatus   string
}

func GenerateGitBoard() (*GitBoard, error) {
   g := &GitBoard{}

   _, err := run("git", "add", ".")
   if err != nil {
      return nil, err
   }
   data, err := run("git", "diff", "--cached", "--numstat")
   if err != nil {
      return nil, err
   }
   lines := strings.FieldsFunc(data, func(r rune) bool {
      return r == '\n'
   })
   for _, line := range lines {
      var add, del int
      fmt.Sscan(line, &add, &del)
      g.Add += add
      g.Delete += del
      g.Change++
   }

   g.Target = 100
   if g.Add >= g.Target {
      g.AddStatus = pass
   } else {
      g.AddStatus = fail
   }
   if g.Delete >= g.Target {
      g.DeleteStatus = pass
   } else {
      g.DeleteStatus = fail
   }
   if g.Change >= g.Target {
      g.ChangeStatus = pass
   } else {
      g.ChangeStatus = fail
   }

   g.Then, err = run("git", "log", "-1", "--format=%cI")
   if err != nil {
      return nil, err
   }
   if len(g.Then) >= 11 {
      g.Then = g.Then[:10]
   }

   g.Now = time.Now().AddDate(0, 0, -1).String()[:10]
   if g.Then <= g.Now {
      g.DateStatus = pass
   } else {
      g.DateStatus = fail
   }

   return g, nil
}
