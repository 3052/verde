package main

import (
   "flag"
   "fmt"
   "net"
   "net/textproto"
   "os"
   "path/filepath"
   "strings"
)

func main() {
   filePath := flag.String("file", "", "local file path to upload")
   phoneIP := flag.String("phoneip", "192.168.158.30", "phone IP address")
   phonePort := flag.String("phoneport", "12345", "phone FTP port")
   pcIP := flag.String("pcip", "192.168.158.166", "PC IP address")
   remoteDir := flag.String("remotedir", "/storage/emulated/0/Music", "remote directory on phone")
   flag.Parse()

   if *filePath == "" {
      fmt.Println("error: -file is required")
      os.Exit(1)
   }

   addr := fmt.Sprintf("%s:%s", *phoneIP, *phonePort)
   conn, err := textproto.Dial("tcp", addr)
   if err != nil {
      panic(err)
   }
   defer conn.Close()

   readReply(conn)
   sendCmd(conn, "USER anonymous")
   sendCmd(conn, "PASS go@script.local")
   sendCmd(conn, "CWD "+*remoteDir)

   err = sendFile(conn, *filePath, *pcIP)
   if err != nil {
      panic(err)
   }
   fmt.Println("done")
}

func readReply(conn *textproto.Conn) string {
   line, err := conn.Reader.ReadLine()
   if err != nil {
      panic(err)
   }
   fmt.Println("<", line)
   return line
}

func sendCmd(conn *textproto.Conn, cmd string) string {
   fmt.Println(">", cmd)
   if err := conn.Writer.PrintfLine("%s", cmd); err != nil {
      panic(err)
   }
   return readReply(conn)
}

func sendFile(conn *textproto.Conn, localPath, pcIP string) error {
   listener, err := net.Listen("tcp", "0.0.0.0:0")
   if err != nil {
      return err
   }
   defer listener.Close()

   port := listener.Addr().(*net.TCPAddr).Port
   p1 := port / 256
   p2 := port % 256
   ipParts := strings.Split(pcIP, ".")
   portCmd := fmt.Sprintf("PORT %s,%s,%s,%s,%d,%d", ipParts[0], ipParts[1], ipParts[2], ipParts[3], p1, p2)
   sendCmd(conn, portCmd)

   sendCmd(conn, fmt.Sprintf("STOR %s", filepath.Base(localPath)))

   dataConn, err := listener.Accept()
   if err != nil {
      return err
   }
   defer dataConn.Close()

   data, err := os.ReadFile(localPath)
   if err != nil {
      return err
   }

   _, err = dataConn.Write(data)
   if err != nil {
      return err
   }
   dataConn.Close()

   readReply(conn)
   return nil
}
