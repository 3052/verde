# magisk

~~~
adb shell monkey -p com.topjohnwu.magisk -c android.intent.category.LAUNCHER 1
~~~

https://topjohnwu.github.io/Magisk/tools.html

~~~
> magisk -apk Magisk-v30.7.apk -img ramdisk.img
19:51:25 === Step 1: Extracting and Preparing Files ===
19:51:25 === Step 2: Pushing Files to Emulator ===
19:51:25 Executing: [adb push Patch_Temp\magisk Patch_Temp\stub.apk Patch_Temp\init-ld Patch_Temp\magiskboot Patch_Temp\magiskinit /data/local/tmp/]
Patch_Temp\magisk: 1 file pushed, 0 skipped. 92.7 MB/s (435344 bytes in 0.004s)
Patch_Temp\stub.apk: 1 file pushed, 0 skipped. 83.2 MB/s (70013 bytes in 0.001s)
Patch_Temp\init-ld: 1 file pushed, 0 skipped. 14.0 MB/s (3984 bytes in 0.000s)
Patch_Temp\magiskboot: 1 file pushed, 0 skipped. 123.3 MB/s (1026536 bytes in 0.008s)
Patch_Temp\magiskinit: 1 file pushed, 0 skipped. 101.9 MB/s (226104 bytes in 0.002s)
5 files pushed, 0 skipped. 31.5 MB/s (1761981 bytes in 0.053s)
19:51:25 Executing: [adb push ramdisk.img /data/local/tmp/ramdisk.img]
ramdisk.img: 1 file pushed, 0 skipped. 198.9 MB/s (1576823 bytes in 0.008s)
19:51:25 === Step 3: Executing CPIO Injection on Emulator ===
19:51:25 Executing adb shell script:
19:51:25   > set -e
19:51:25   > cd /data/local/tmp
19:51:25   > chmod +x magiskboot
19:51:25   > ./magiskboot decompress ramdisk.img ramdisk.cpio
19:51:25   > ./magiskboot cpio ramdisk.cpio 'mkdir 0750 overlay.d' 'mkdir 0750 overlay.d/sbin' 'mkdir 0000 .backup' 'mv init .backup/init' 'add 0644 overlay.d/sbin/stub.apk stub.apk' 'add 0750 init magiskinit' 'add 0755 overlay.d/sbin/init-ld init-ld' 'add 0755 overlay.d/sbin/magisk magisk'
19:51:25   > ./magiskboot compress=lz4_legacy ramdisk.cpio magisk_patched.img
Detected format: lz4_legacy
Loading cpio: [ramdisk.cpio]
Create directory [overlay.d] (0750)
Create directory [overlay.d/sbin] (0750)
Create directory [.backup] (0000)
Move [init] -> [.backup/init]
Add file [overlay.d/sbin/stub.apk] (100644)
Add file [init] (100750)
Add file [overlay.d/sbin/init-ld] (100755)
Add file [overlay.d/sbin/magisk] (100755)
Dumping cpio: [ramdisk.cpio]
19:51:27 === Step 4: Pulling Patched Image ===
19:51:27 Executing: [adb pull /data/local/tmp/magisk_patched.img .]
/data/local/tmp/magisk_patched.img: 1 file pulled, 0 skipped. 78.5 MB/s (2015998 bytes in 0.025s)
~~~

## Step 4: Flash the Emulator & Wipe Data

1. Completely close the running TV emulator.
2. Go to your Windows SDK folder (`%LOCALAPPDATA%\Android\Sdk\system-images\android-34\android-tv\x86\`).
3. Rename the original `ramdisk.img` to `ramdisk.img.stock` (to keep as a backup).
4. Move your new `magisk_patched.img` from your `Patch` folder into this SDK folder.
5. Rename `magisk_patched.img` to exactly `ramdisk.img`.
6. Open Android Studio Device Manager. Click the **three dots** next to your TV and click **Wipe Data**. (Crucial: Skipping this causes a permanent black screen).
7. Click the **three dots** again and click **Cold Boot Now**.

## Step 5: Finalize Magisk

1. Once the TV reaches the home screen, install the Magisk app from your
   PowerShell window (rename your zip back to apk first):
   `adb install Magisk-v30.7.apk`
2. Open the Magisk app on the TV. It will say Installed 30.7. 
3. your device needs additional setup for Magisk to work properly. Do you want
   to proceed and reboot? OK
4. Navigate to the **Superuser** tab (the shield icon at the bottom).
5. Toggle the switch ON for **[SharedUID] Shell (com.android.shell)**.

You now have unrestricted `su` root access on an API 34 Android TV emulator.
Install your ARM APK, launch it, and use the Superuser tab to grant it root
permissions.
