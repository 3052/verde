package main

import (
   "flag"
   "fmt"
   "log"
   "os"
   "os/exec"
)

func main() {
   err := new(command).run()
   if err != nil {
      log.Fatal(err)
   }
}

type command struct {
   all      bool
   duration string
   ext      string
   input    string
   start    string
}

func (c *command) do_input() error {
   arg := []string{"-hide_banner"}
   if c.start != "" {
      arg = append(arg, "-ss", c.start)
   }
   arg = append(arg, "-i", c.input)
   if c.duration != "" {
      arg = append(arg, "-t", c.duration)
   }
   arg = append(arg, "-q", "1", "-vsync", "vfr")
   if !c.all {
      arg = append(arg, "-vf", "select='eq(pict_type, I)'")
   }
   arg = append(arg, "%d"+c.ext)
   ffmpeg := exec.Command("ffmpeg", arg...)
   ffmpeg.Stderr = os.Stderr
   log.Print(ffmpeg)
   return ffmpeg.Run()
}

func (c *command) run() error {
   exts := []string{".jpg", ".png"}
   flag.BoolVar(&c.all, "a", false, "output all frames")
   flag.StringVar(&c.duration, "d", "", "duration")
   flag.StringVar(&c.input, "i", "", "input file")
   flag.StringVar(&c.start, "s", "", "start")
   flag.StringVar(&c.ext, "e", exts[0], fmt.Sprint(exts))
   flag.Parse()
   if c.input != "" {
      return c.do_input()
   }
   flag.Usage()
   return nil
}
