; Inno Setup script for Drive (Windows). Build on Windows with Inno Setup 6:
;   iscc /DAppVersion=1.2.3 /DSourceDir=..\..\build\out\windows-amd64 /DOutputDir=..\..\dist drive.iss
; SourceDir holds drive.exe and drive.ico (scripts/package.sh puts them there).

#ifndef AppVersion
  #define AppVersion "0.0.0"
#endif
#ifndef SourceDir
  #define SourceDir "..\..\build\out\windows-amd64"
#endif
#ifndef OutputDir
  #define OutputDir "..\..\dist"
#endif

[Setup]
AppId={{5B0F6C3E-7D1A-4C52-9C1E-3A6F2D8B9E41}
AppName=Drive
AppVersion={#AppVersion}
AppPublisher=Drive for Urbit
DefaultDirName={autopf}\Drive
DefaultGroupName=Drive
DisableProgramGroupPage=yes
OutputDir={#OutputDir}
OutputBaseFilename=drive-{#AppVersion}-windows-amd64-setup
SetupIconFile={#SourceDir}\drive.ico
UninstallDisplayIcon={app}\drive.exe
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
Compression=lzma2
SolidCompression=yes
WizardStyle=modern
CloseApplications=yes

[Tasks]
Name: "startup"; Description: "Start Drive when I log in"; GroupDescription: "Startup:"

[Files]
Source: "{#SourceDir}\drive.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#SourceDir}\drive.ico"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{group}\Drive"; Filename: "{app}\drive.exe"; Parameters: "tray"; IconFilename: "{app}\drive.ico"
Name: "{group}\Uninstall Drive"; Filename: "{uninstallexe}"

[Registry]
; the same value the tray's "Start at login" toggle sets (internal/ui/autostart_windows.go)
Root: HKCU; Subkey: "Software\Microsoft\Windows\CurrentVersion\Run"; ValueType: string; \
  ValueName: "Drive"; ValueData: """{app}\drive.exe"" ""tray"""; Flags: uninsdeletevalue; Tasks: startup

[Run]
Filename: "{app}\drive.exe"; Parameters: "tray"; Description: "Start Drive"; Flags: nowait postinstall skipifsilent

[UninstallRun]
Filename: "{sys}\taskkill.exe"; Parameters: "/IM drive.exe /F"; Flags: runhidden; RunOnceId: "StopDrive"
