package main

import "fmt"

var tests = []test{
   {
      go_sum:    0,
      issue:     "tag version",
      issue_url: "github.com/tadvi/winc/issues/14",
      linux:     false,
      pushed_at: 2021,
      size:      "1.62 MB",
   },
   {
      go_sum:    115,
      issue:     "smaller backend",
      issue_url: "github.com/aarzilli/nucular/issues/89",
      linux:     true,
      size:      "2.75 MB",
   },
   {
      go_sum:    0,
      linux:     false,
      pushed_at: 2023,
      size:      "1.69 MB",
      issue_url: "github.com/rodrigocfd/windigo/issues/38",
      issue:     "export option types",
   },
   {
      size:      "8.44 MB",
      issue_url: "github.com/yottahmd/furex/issues/74",
      issue:     "executable size",
   },
   {
      size:      "8.93 MB",
      issue_url: "github.com/demouth/ebitenlg/issues/1",
      issue:     "executable size",
   },
   {
      issue:     "executable size",
      issue_url: "github.com/hajimehoshi/guigui/issues/27",
      size:      "10.4 MB",
   },
   {
      url:       "github.com/zeozeozeo/ebitengine-microui-go",
      issue_url: "github.com/zeozeozeo/microui-go/issues/4",
      issue:     "smaller demo",
      size:      "10.6 MB",
   },
   {
      issue_url: "github.com/zeozeozeo/microui-go/issues/4",
      size:      "10.6 MB",
      issue:     "smaller demo",
      url:       "github.com/zeozeozeo/microui-go",
   },
   {
      size:      "15.7 MB",
      issue_url: "codeberg.org/tslocum/etk/issues/11",
      issue:     "executable size is large",
   },
   {
      issue_url: "github.com/cogentcore/core/issues/1502",
      issue:     "wgpuglfw: build constraints exclude all Go files",
   },
   {
      issue:     "build without C",
      issue_url: "github.com/richardwilkes/unison/issues/64",
   },
   {
      issue:     "build without C",
      issue_url: "github.com/fyne-io/fyne/issues/5651",
   },
   {
      issue:     "make it clear that this is not pure Go",
      issue_url: "github.com/AllenDang/giu/issues/965",
   },
   {
      issue:     "undefined: syscall.Mkfifo",
      issue_url: "github.com/codeation/impress/issues/3",
   },
   {
      issue:     "static library",
      issue_url: "github.com/twgh/xcgui/issues/41",
   },
   {
      issue:     "BindWidgets error GetDlgItem",
      issue_url: "github.com/whtiehack/wingui/issues/18",
   },
   {
      issue_url: "github.com/ying32/govcl/issues/226",
      issue:     "static Liblcl",
   },
   {
      issue_url: "github.com/AllenDang/gform/issues/18",
      issue:     "cannot use flag (type uint32)",
   },
   {
      issue_url: "github.com/torlangballe/zui/issues/4",
      issue:     "found packages zwindow (Window!js.go) and zui (Window_windows.go)",
   },
}

func main() {
   fmt.Println(len(tests))
}

type test struct {
   go_sum    int
   issue     string
   issue_url string
   linux     bool
   pushed_at int
   size      string
   url       string
}
