package main

import (
   "archive/zip"
   "bytes"
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

// ---------------------------------------------------------------------------
// Zip extraction helpers
// ---------------------------------------------------------------------------

func extractFromZip(zipPath string, filesToExtract map[string]string, destDir string) error {
   reader, err := zip.OpenReader(zipPath)
   if err != nil {
      return err
   }
   defer reader.Close()

   foundCount := 0
   for _, file := range reader.File {
      if destName, wantsFile := filesToExtract[file.Name]; wantsFile {
         if err := extractSingleFile(file, filepath.Join(destDir, destName)); err != nil {
            return err
         }
         foundCount++
      }
   }

   if foundCount != len(filesToExtract) {
      return fmt.Errorf("could not find all required files in the APK. Make sure you downloaded the full Magisk APK")
   }
   return nil
}

func extractSingleFile(file *zip.File, dest string) error {
   rc, err := file.Open()
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

// initLdApkPath validates the bitness and returns the corresponding
// APK-internal path for the init-ld library.
func initLdApkPath(initBits int) (string, error) {
   switch initBits {
   case 32:
      return "lib/x86/libinit-ld.so", nil
   case 64:
      return "lib/x86_64/libinit-ld.so", nil
   default:
      return "", fmt.Errorf("invalid -init-bits value %d (want 32 or 64)", initBits)
   }
}

// ---------------------------------------------------------------------------
// main
// ---------------------------------------------------------------------------

func main() {
   log.SetFlags(log.Ltime)

   apkPath := flag.String("apk", "", "Path to the Magisk APK file (e.g., Magisk-v30.7.apk)")
   ramdiskPath := flag.String("img", "", "Path to the unpatched ramdisk.img file")
   initBits := flag.Int("init-bits", 0,
      `Required. Bitness of /system/bin/init to match for init-ld preload. Values: 32 or 64.`)

   flag.Parse()

   if *apkPath == "" || *ramdiskPath == "" || *initBits == 0 {
      flag.PrintDefaults()
      log.Fatal("Error: -apk, -img, and -init-bits flags are all required.")
   }

   if err := performPatch(*apkPath, *ramdiskPath, *initBits); err != nil {
      log.Fatalf("Patching failed: %v", err)
   }
}

func performPatch(apkPath string, ramdiskPath string, initBits int) error {
   patchDir := "Patch_Temp"

   log.Print("=== Step 1: Extracting and Preparing Files ===")

   if err := os.MkdirAll(patchDir, 0755); err != nil {
      return fmt.Errorf("error creating temp dir: %w", err)
   }
   defer os.RemoveAll(patchDir)

   initLdPath, err := initLdApkPath(initBits)
   if err != nil {
      return err
   }
   log.Printf("[config] using %s for init-ld", initLdPath)

   // Dynamically select the magisk daemon bitness to match the init bitness
   var magiskDaemonApkPath string
   var magiskDaemonName string

   if initBits == 32 {
      magiskDaemonApkPath = "lib/x86/libmagisk.so"
      magiskDaemonName = "magisk32"
   } else {
      magiskDaemonApkPath = "lib/x86_64/libmagisk.so"
      magiskDaemonName = "magisk"
   }
   log.Printf("[config] using %s for magisk daemon", magiskDaemonName)

   filesToExtract := map[string]string{
      "lib/x86_64/libmagiskboot.so": "magiskboot",
      "lib/x86_64/libmagiskinit.so": "magiskinit",
      "assets/stub.apk":             "stub.apk",
      initLdPath:                    "init-ld",
      magiskDaemonApkPath:           magiskDaemonName,
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

   // --- Get PREINITDEVICE from the magisk binary running on the emulator ---
   log.Print("=== Step 2b: Detecting PREINITDEVICE ===")
   preinitDevice, err := runAdbShellCapture(
      "cd /data/local/tmp",
      fmt.Sprintf("chmod +x %s", magiskDaemonName),
      fmt.Sprintf("./%s --preinit-device", magiskDaemonName),
   )
   if err != nil {
      return fmt.Errorf("failed to get preinit device: %w", err)
   }
   log.Printf("[config] PREINITDEVICE=%s", preinitDevice)

   log.Print("=== Step 3: Executing CPIO Injection on Emulator ===")
   cpioArgs := []string{
      "./magiskboot cpio ramdisk.cpio",
      "'mkdir 0750 overlay.d'",
      "'mkdir 0750 overlay.d/sbin'",
      "'add 0750 init magiskinit'",
      // Files MUST be xz-compressed — extract_files() in magiskinit
      // only looks for *.xz files, not raw binaries.
      "'add 0644 overlay.d/sbin/magisk.xz magisk.xz'",
      "'add 0644 overlay.d/sbin/stub.xz stub.xz'",
      "'add 0644 overlay.d/sbin/init-ld.xz init-ld.xz'",
      "'patch'",
      "'backup ramdisk.cpio.orig'",
      "'mkdir 000 .backup'",
      "'add 000 .backup/.magisk config'",
   }

   if err := runAdbShell(
      "cd /data/local/tmp",
      "chmod +x magiskboot",
      // Write config with PREINITDEVICE
      "echo KEEPVERITY=true > config",
      "echo KEEPFORCEENCRYPT=true >> config",
      "echo RECOVERYMODE=false >> config",
      fmt.Sprintf("echo PREINITDEVICE=%s >> config", preinitDevice),
      // Compress files with xz BEFORE adding to cpio
      fmt.Sprintf("./magiskboot compress=xz %s magisk.xz", magiskDaemonName),
      "./magiskboot compress=xz stub.apk stub.xz",
      "./magiskboot compress=xz init-ld init-ld.xz",
      // Decompress ramdisk
      "./magiskboot decompress ramdisk.img ramdisk.cpio",
      // Copy for the "backup" patch to succeed
      "cp ramdisk.cpio ramdisk.cpio.orig",
      strings.Join(cpioArgs, " "),
      // Always use gzip for final output
      "./magiskboot compress=gzip ramdisk.cpio magisk_patched.img",
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

func run(name string, arg ...string) error {
   cmd := exec.Command(name, arg...)
   log.Printf("Executing: %v", cmd.Args)
   cmd.Stdout = os.Stdout
   cmd.Stderr = os.Stderr
   return cmd.Run()
}

func runAdbShell(scripts ...string) error {
   scripts = slices.Insert(scripts, 0, "set -e")

   log.Println("Executing adb shell script:")
   for _, line := range scripts {
      log.Printf("  > %s", line)
   }

   cmd := exec.Command("adb", "shell")
   cmd.Stdin = strings.NewReader(strings.Join(scripts, "\n"))
   cmd.Stdout = os.Stdout
   cmd.Stderr = os.Stderr
   return cmd.Run()
}

// runAdbShellCapture runs commands via adb shell and returns stdout.
func runAdbShellCapture(scripts ...string) (string, error) {
   scripts = slices.Insert(scripts, 0, "set -e")

   log.Println("Executing adb shell script (capture):")
   for _, line := range scripts {
      log.Printf("  > %s", line)
   }

   cmd := exec.Command("adb", "shell")
   cmd.Stdin = strings.NewReader(strings.Join(scripts, "\n"))
   var out bytes.Buffer
   cmd.Stdout = &out
   cmd.Stderr = os.Stderr
   if err := cmd.Run(); err != nil {
      return "", err
   }
   return strings.TrimSpace(out.String()), nil
}
