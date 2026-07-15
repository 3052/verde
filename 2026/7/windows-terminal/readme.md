# windows terminal

DON'T download the MsixBundle, it does stupid stuff with the installation. the
terminal gets installed to some weird folder with extreme security settings. I
cant mess with it even as an admin. instead just get the dumb zip file, like
this:

<https://github.com/microsoft/terminal/releases/download/v1.18.2822.0/Microsoft.WindowsTerminal_1.18.2822.0_x64.zip>

extract to wherever, for example:

~~~
C:\terminal-1.18.2822.0
~~~

then save the below as INSTALL.REG:

~~~
Windows Registry Editor Version 5.00
[HKEY_CLASSES_ROOT\Directory\background\shell\terminal\command]
@="C:\\terminal-1.18.2822.0\\WindowsTerminal.exe"
~~~

and you can fix the starting folder like this:

~~~json
{
   "profiles": {
      "defaults": {"startingDirectory": ""}
   }
}
~~~

## wezterm

11 MB:

https://github.com/microsoft/terminal/releases/tag/v1.20.11781.0

63 MB:

https://github.com/wez/wezterm/releases/tag/20240203-110809-5046fc22

## scrollToTop

https://devblogs.microsoft.com/commandline/windows-terminal-preview-1-6-release

## find

https://devblogs.microsoft.com/commandline/windows-terminal-preview-v0-8-release
