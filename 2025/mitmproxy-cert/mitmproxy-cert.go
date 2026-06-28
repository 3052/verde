package main

import (
   "crypto/md5"
   "crypto/x509"
   "encoding/hex"
   "encoding/pem"
   "flag"
   "fmt"
   "log"
   "os"
   "os/exec"
   "path/filepath"
)

const (
   from = "/data/local/tmp/cacerts"
   to   = "/system/etc/security/cacerts"
)

func main() {
   var info bool
   flag.BoolVar(&info, "i", false, "information")
   cert, err := os.UserHomeDir()
   if err != nil {
      log.Fatal(err)
   }
   cert = filepath.ToSlash(cert) + "/.mitmproxy/mitmproxy-ca-cert.pem"
   flag.StringVar(&cert, "c", cert, "certificate")
   flag.Parse()
   data, err := os.ReadFile(cert)
   if err != nil {
      log.Fatal(err)
   }
   data, err = subject_hash(data)
   if err != nil {
      log.Fatal(err)
   }
   push := hex.EncodeToString(data) + ".0"
   commands := [][]string{
      {"adb", "shell", "mkdir", "-p", from},
      {"adb", "shell", "cp", to + "/*", from},
      {"adb", "push", cert, from + "/" + push},
      {"adb", "root"},
      {"adb", "wait-for-device"},
      {"adb", "shell", "mount", "-t", "tmpfs", "tmpfs", to},
      // mv fails with Android API 18
      {"adb", "shell", "cp", from + "/*", to},
      {"adb", "shell", "chcon", "u:object_r:system_file:s0", to + "/*"},
   }
   for _, command := range commands {
      cmd := exec.Command(command[0], command[1:]...)
      cmd.Stderr = os.Stderr
      cmd.Stdout = os.Stdout
      fmt.Println(cmd.Args)
      if !info {
         err := cmd.Run()
         if err != nil {
            log.Fatal(err)
         }
      }
   }
}

// outputs the MD5 "hash" of the certificate subject name
func subject_hash(data []byte) ([]byte, error) {
   block, _ := pem.Decode(data)
   cert, err := x509.ParseCertificate(block.Bytes)
   if err != nil {
      return nil, err
   }
   sum := md5.Sum(cert.RawSubject)
   return []byte{sum[3], sum[2], sum[1], sum[0]}, nil
}
