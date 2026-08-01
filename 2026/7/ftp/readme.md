# Phone File Transfer (FTP over USB Tethering)

## Part 1: One-time phone setup

### 1. Install Primitive FTPd
- On the phone, open a browser and go to https://github.com/wolpi/prim-ftpd/releases
- Download the latest `.apk` file
- Tap the downloaded file, tap **Install**, accept the "unknown source" warning

### 2. Enable USB debugging
- Go to **Settings → About phone**
- Tap the **search bar** at the top, type **build number**
- Tap the **Build number** result 7 times until you see "You are now a developer"
- Go back to **Settings**, tap the search bar, type **developer options**
- Tap the result, toggle **USB debugging ON**, accept the warning

### 3. Configure Primitive FTPd
- Open **Primitive FTPd**
- Tap the **gear icon** (Preferences)
- If prompted to generate keys, tap **Generate**
- Set **anonymous login** → **true**
- Go back to the main screen

### 4. Grant file access
- Go to **Settings → Apps → Special app access → All files access**
- Turn on **Primitive FTPd**

## Part 2: Every time you want to transfer a file

### 1. Plug phone into PC via USB

### 2. Enable USB tethering
- On phone: **Settings → Network & internet → Hotspot & tethering → USB tethering** → ON

### 3. Find the PC's IP
- On PC, open PowerShell, run `ipconfig`
- Look for **Ethernet 2** (or whichever adapter has IP `192.168.158.x`)
- Note the PC's IP (e.g., `192.168.158.166`)

### 4. Start the FTP server
- Open **Primitive FTPd** on the phone
- Tap **start/play**
- Tap **rndis0**
- Note the IP (e.g., `192.168.158.30`) and FTP port (e.g., `12345`)

### 5. Run the Go script
```
go run main.go -file D:\Desktop\hello.txt -phoneip 192.168.158.30 -phoneport 12345 -pcip 192.168.158.166 -remotedir /storage/emulated/0/Music
```

## Part 3: When done

1. Tap **stop** in Primitive FTPd
2. Turn off USB tethering
3. Unplug phone

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-file` | (required) | Local file to upload |
| `-phoneip` | `192.168.158.30` | Phone IP (from Primitive FTPd) |
| `-phoneport` | `12345` | Phone FTP port (from Primitive FTPd) |
| `-pcip` | `192.168.158.166` | PC IP (from `ipconfig`, Ethernet 2) |
| `-remotedir` | `/storage/emulated/0/Music` | Remote directory on phone |

## Example

```
go run main.go -file D:\Desktop\hello.txt
```

(Defaults match your current setup — just change `-file` each time.)
