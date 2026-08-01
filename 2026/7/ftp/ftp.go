package main

import (
   "flag"
   "fmt"
   "net"
   "net/textproto"
   "os"
   "path/filepath"
   "strings"
   "sync"
   "time"
)

const phonePort = "12345"

func findPhone() (string, string) {
   ifaces, _ := net.Interfaces()
   for _, iface := range ifaces {
      for _, addr := range iface.Addrs() {
         ipNet, ok := addr.(*net.IPNet)
         if !ok || ipNet.IP.To4() == nil || ipNet.IP.IsLoopback() {
            continue
         }
         pcIP := ipNet.IP.String()
         if phoneIP := scanSubnet(pcIP); phoneIP != "" {
            return pcIP, phoneIP
         }
      }
   }
   panic("phone not found")
}

func main() {
   filePath := flag.String("file", "", "local file path to upload")
   remoteDir := flag.String("remotedir", "/storage/emulated/0/Music", "remote directory on phone")
   flag.Parse()
   if *filePath == "" {
      fmt.Println("error: -file is required")
      os.Exit(1)
   }
   pcIP, phoneIP := findPhone()
   fmt.Printf("PC: %s, Phone: %s\n", pcIP, phoneIP)
   conn, err := textproto.Dial("tcp", phoneIP+":"+phonePort)
   if err != nil {
      panic(err)
   }
   defer conn.Close()
   readReply(conn)
   sendCmd(conn, "USER anonymous")
   sendCmd(conn, "PASS go@script.local")
   sendCmd(conn, "CWD "+*remoteDir)
   if err := sendFile(conn, *filePath, pcIP); err != nil {
      panic(err)
   }
   fmt.Println("done")
}

func readReply(conn *textproto.Conn) {
   line, err := conn.Reader.ReadLine()
   if err != nil {
      panic(err)
   }
   fmt.Println("<", line)
}

func scanSubnet(pcIP string) string {
   parts := strings.Split(pcIP, ".")
   prefix := parts[0] + "." + parts[1] + "." + parts[2] + "."
   var wg sync.WaitGroup
   found := make(chan string, 1)
   for i := 1; i < 255; i++ {
      target := fmt.Sprintf("%s%d", prefix, i)
      if target == pcIP {
         continue
      }
      wg.Add(1)
      go func(t string) {
         defer wg.Done()
         c, err := net.DialTimeout("tcp", t+":"+phonePort, 500*time.Millisecond)
         if err != nil {
            return
         }
         c.Close()
         select {
         case found <- t:
         default:
         }
      }(target)
   }
   go func() { wg.Wait(); close(found) }()
   if phoneIP, ok := <-found; ok {
      return phoneIP
   }
   return ""
}

func sendCmd(conn *textproto.Conn, cmd string) {
   fmt.Println(">", cmd)
   conn.Writer.PrintfLine("%s", cmd)
   readReply(conn)
}

func sendFile(conn *textproto.Conn, localPath, pcIP string) error {
   listener, err := net.Listen("tcp", "0.0.0.0:0")
   if err != nil {
      return err
   }
   defer listener.Close()
   port := listener.Addr().(*net.TCPAddr).Port
   p := strings.Split(pcIP, ".")
   sendCmd(conn, fmt.Sprintf("PORT %s,%s,%s,%s,%d,%d", p[0], p[1], p[2], p[3], port/256, port%256))
   sendCmd(conn, "STOR "+filepath.Base(localPath))
   dataConn, err := listener.Accept()
   if err != nil {
      return err
   }
   data, err := os.ReadFile(localPath)
   if err != nil {
      dataConn.Close()
      return err
   }
   dataConn.Write(data)
   dataConn.Close()
   readReply(conn)
   return nil
}
