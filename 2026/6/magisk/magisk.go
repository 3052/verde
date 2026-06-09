package main

import (
   "archive/zip"
   "fmt"
   "io"
   "os"
   "os/exec"
   "path/filepath"
)

func main() {
   if len(os.Args) < 3 {
      fmt.Println("Usage: patcher <path_to_magisk_apk> <path_to_ramdisk_img>")
      os.Exit(1)
   }

   apkPath := os.Args[1]
   ramdiskPath := os.Args[2]
   patchDir := "Patch_Temp"

   fmt.Println("=== Step 1: Extracting and Preparing Files ===")

   err := os.MkdirAll(patchDir, 0755)
   if err != nil {
      fmt.Printf("Error creating temp dir: %v\n", err)
      os.Exit(1)
   }
   defer os.RemoveAll(patchDir) // Clean up temp files when done

   filesToExtract := map[string]string{
      "lib/x86/libmagiskboot.so": "magiskboot",
      "lib/x86/libmagiskinit.so": "magiskinit",
      "lib/x86/libmagisk.so":     "magisk32",
      "lib/x86/libinit-ld.so":    "init-ld",
      "assets/stub.apk":          "stub.apk",
   }

   err = extractFromZip(apkPath, filesToExtract, patchDir)
   if err != nil {
      fmt.Printf("Error extracting from APK: %v\n", err)
      os.Exit(1)
   }

   err = copyFile(ramdiskPath, filepath.Join(patchDir, "ramdisk.img"))
   if err != nil {
      fmt.Printf("Error copying ramdisk.img: %v\n", err)
      os.Exit(1)
   }
   fmt.Println("Files prepared successfully.")

   fmt.Println("\n=== Step 2: Pushing Files to Emulator ===")

   runAdb("shell", "mkdir -p /data/local/tmp/patch")

   pushArgs := []string{"push"}
   for _, destName := range filesToExtract {
      pushArgs = append(pushArgs, filepath.Join(patchDir, destName))
   }
   pushArgs = append(pushArgs, filepath.Join(patchDir, "ramdisk.img"), "/data/local/tmp/patch/")
   runAdb(pushArgs...)

   runAdb("shell", "chmod +x /data/local/tmp/patch/magiskboot")

   fmt.Println("\n=== Step 3: Executing CPIO Injection on Emulator ===")

   fmt.Println("Decompressing ramdisk...")
   runAdb("shell", "cd /data/local/tmp/patch && ./magiskboot decompress ramdisk.img ramdisk.cpio")

   fmt.Println("Creating Magisk config...")
   runAdb("shell", "cd /data/local/tmp/patch && printf 'KEEPVERITY=true\\nKEEPFORCEENCRYPT=true\\nRECOVERYMODE=false\\n' > config")

   fmt.Println("Injecting binaries and hijacking boot sequence...")
   cpioCmd := `cd /data/local/tmp/patch && ./magiskboot cpio ramdisk.cpio "mkdir 0000 .backup" "mv init .backup/init" "add 0000 .backup/.magisk config" "mkdir 0750 overlay.d" "mkdir 0750 overlay.d/sbin" "add 0750 init magiskinit" "add 0755 overlay.d/sbin/magisk magisk32" "add 0755 overlay.d/sbin/magisk32 magisk32" "add 0755 overlay.d/sbin/magisk64 magisk32" "add 0755 overlay.d/sbin/init-ld init-ld" "add 0644 overlay.d/sbin/stub.apk stub.apk"`
   runAdb("shell", cpioCmd)

   fmt.Println("Compressing patched ramdisk...")
   runAdb("shell", "cd /data/local/tmp/patch && ./magiskboot compress=lz4_legacy ramdisk.cpio magisk_patched.img")

   fmt.Println("Pulling patched file to current directory...")
   runAdb("pull", "/data/local/tmp/patch/magisk_patched.img", ".")

   fmt.Println("\n=== Cleaning Up Emulator Temp Files ===")
   runAdb("shell", "rm -rf /data/local/tmp/patch")

   fmt.Println("\nSUCCESS! You can now move magisk_patched.img to your SDK folder and cold boot the emulator.")
}

func runAdb(args ...string) {
   cmd := exec.Command("adb", args...)
   cmd.Stdout = os.Stdout
   cmd.Stderr = os.Stderr
   err := cmd.Run()
   if err != nil {
      fmt.Printf("ADB command failed: adb %v\n", args)
   }
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
         err := extractSingleFile(f, filepath.Join(destDir, destName))
         if err != nil {
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

func copyFile(src, dst string) error {
   sourceFile, err := os.Open(src)
   if err != nil {
      return err
   }
   defer sourceFile.Close()

   destFile, err := os.Create(dst)
   if err != nil {
      return err
   }
   defer destFile.Close()

   _, err = io.Copy(destFile, sourceFile)
   return err
}
