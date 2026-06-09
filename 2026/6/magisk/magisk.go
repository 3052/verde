package main

import (
   "archive/zip"
   "flag"
   "fmt"
   "io"
   "log"
   "os"
   "os/exec"
   "path/filepath"
   "slices"
   "strings"
)

func performPatch(apkPath string, ramdiskPath string) error {
   patchDir := "Patch_Temp"

   log.Print("=== Step 1: Extracting and Preparing Files ===")

   if err := os.MkdirAll(patchDir, 0755); err != nil {
      return fmt.Errorf("error creating temp dir: %w", err)
   }
   defer os.RemoveAll(patchDir)
   filesToExtract := map[string]string{
      // will not boot:
      "lib/x86/libmagiskboot.so": "magiskboot",
      "lib/x86/libmagiskinit.so": "magiskinit",
      "lib/x86/libmagisk.so":     "magisk",
      // will boot but Magisk will not load:
      "assets/stub.apk": "stub.apk",
      // will boot but no root:
      "lib/x86/libinit-ld.so": "init-ld",
   }
   if err := extractFromZip(apkPath, filesToExtract, patchDir); err != nil {
      return fmt.Errorf("error extracting from APK: %w", err)
   }
   log.Print("=== Step 2: Pushing Files to Emulator ===")
   pushArgs := []string{"push"}
   for _, destName := range filesToExtract {
      pushArgs = append(pushArgs, filepath.Join(patchDir, destName))
   }
   pushArgs = append(pushArgs, "/data/local/tmp/")
   if err := run("adb", pushArgs...); err != nil {
      return err
   }
   if err := run("adb", "push", ramdiskPath, "/data/local/tmp/ramdisk.img"); err != nil {
      return err
   }
   log.Print("=== Step 3: Executing CPIO Injection on Emulator ===")
   cpioArgs := []string{
      "./magiskboot cpio ramdisk.cpio",              // KEEP
      "'mkdir 0750 overlay.d'",                      // KEEP
      "'mkdir 0750 overlay.d/sbin'",                 // KEEP
      "'mkdir 0000 .backup'",                        // KEEP
      "'mv init .backup/init'",                      // KEEP
      "'add 0644 overlay.d/sbin/stub.apk stub.apk'", // KEEP
      "'add 0750 init magiskinit'",                  // KEEP
      "'add 0755 overlay.d/sbin/init-ld init-ld'",   // KEEP
      "'add 0755 overlay.d/sbin/magisk magisk'",
   }
   if err := runAdbShell(
      "cd /data/local/tmp",
      "chmod +x magiskboot",
      "./magiskboot decompress ramdisk.img ramdisk.cpio",
      strings.Join(cpioArgs, " "),
      "./magiskboot compress=lz4_legacy ramdisk.cpio magisk_patched.img",
   ); err != nil {
      return err
   }
   log.Print("=== Step 4: Pulling Patched Image ===")

   if err := run("adb", "pull", "/data/local/tmp/magisk_patched.img", "."); err != nil {
      return err
   }

   log.Print("SUCCESS! You can now move magisk_patched.img to your SDK folder and cold boot the emulator.")
   return nil
}

func runAdbShell(scripts ...string) error {
   scripts = slices.Insert(scripts, 0, "set -e")

   log.Println("Executing adb shell script:")
   for _, s := range scripts {
      log.Printf("  > %s", s)
   }

   cmd := exec.Command("adb", "shell")
   cmd.Stdin = strings.NewReader(strings.Join(scripts, "\n"))
   cmd.Stdout = os.Stdout
   cmd.Stderr = os.Stderr
   return cmd.Run()
}

func run(name string, arg ...string) error {
   cmd := exec.Command(name, arg...)
   log.Printf("Executing: %v", cmd.Args)
   cmd.Stdout = os.Stdout
   cmd.Stderr = os.Stderr
   return cmd.Run()
}

func extractFromZip(zipPath string, filesToExtract map[string]string, destDir string) error {
   r, err := zip.OpenReader(zipPath)
   if err != nil {
      return err
   }
   defer r.Close()

   foundCount := 0
   for _, f := range r.File {
      if destName, wantsFile := filesToExtract[f.Name]; wantsFile {
         if err := extractSingleFile(f, filepath.Join(destDir, destName)); err != nil {
            return err
         }
         foundCount++
      }
   }

   if foundCount != len(filesToExtract) {
      return fmt.Errorf("could not find all required x86 files in the APK. Make sure you downloaded the full Magisk APK")
   }
   return nil
}

func extractSingleFile(f *zip.File, dest string) error {
   rc, err := f.Open()
   if err != nil {
      return err
   }
   defer rc.Close()

   destFile, err := os.Create(dest)
   if err != nil {
      return err
   }
   defer destFile.Close()

   _, err = io.Copy(destFile, rc)
   return err
}

func main() {
   log.SetFlags(log.Ltime)
   apkPath := flag.String("apk", "", "Path to the Magisk APK file (e.g., Magisk-v30.7.apk)")
   ramdiskPath := flag.String("img", "", "Path to the unpatched ramdisk.img file")

   flag.Parse()

   if *apkPath == "" || *ramdiskPath == "" {
      flag.PrintDefaults()
      log.Fatal("Error: Both -apk and -img flags are required.")
   }

   if err := performPatch(*apkPath, *ramdiskPath); err != nil {
      log.Fatalf("Patching failed: %v", err)
   }
}
