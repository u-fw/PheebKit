; Build only through scripts/release-windows.ps1, from one verified payload.
#ifndef AppVersion
  #error AppVersion must be supplied by the release script
#endif
#ifndef WindowsVersion
  #error WindowsVersion must be supplied by the release script
#endif
#ifndef PayloadDir
  #error PayloadDir must be supplied by the release script
#endif
#ifndef ReleaseDir
  #error ReleaseDir must be supplied by the release script
#endif

[Setup]
AppId={{9D8FAD06-8029-4A58-9497-96C7DA878C63}
AppName=菲比工具箱 (PheebKit)
AppVersion={#AppVersion}
AppVerName=菲比工具箱 (PheebKit) {#AppVersion}
VersionInfoVersion={#WindowsVersion}
VersionInfoTextVersion={#AppVersion}
VersionInfoProductTextVersion={#AppVersion}
DefaultDirName={localappdata}\Programs\PheebKit
DefaultGroupName=菲比工具箱 (PheebKit)
PrivilegesRequired=lowest
ArchitecturesAllowed=x64os
ArchitecturesInstallIn64BitMode=x64os
MinVersion=10.0
OutputDir={#ReleaseDir}
OutputBaseFilename=PheebKit-{#AppVersion}-windows-x64-setup
SetupIconFile=icon.ico
UninstallDisplayIcon={app}\PheebKit.exe
LicenseFile={#PayloadDir}\LICENSE.txt
Compression=lzma2
SolidCompression=yes
WizardStyle=modern
AllowNoIcons=yes
DisableProgramGroupPage=no
CloseApplications=no
RestartApplications=no
AppMutex=Local\kompeg-application-running

[Languages]
Name: "chinesesimplified"; MessagesFile: "compiler:Languages\ChineseSimplified.isl"
Name: "english"; MessagesFile: "compiler:Default.isl"

[Tasks]
Name: "desktopicon"; Description: "{cm:CreateDesktopIcon}"; Flags: unchecked

[Files]
Source: "{#PayloadDir}\PheebKit.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#PayloadDir}\LICENSE.txt"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#PayloadDir}\THIRD_PARTY_NOTICES.txt"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#PayloadDir}\README.txt"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#PayloadDir}\RELEASE_NOTES.md"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#PayloadDir}\BUILD_INFO.json"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{group}\菲比工具箱 (PheebKit)"; Filename: "{app}\PheebKit.exe"
Name: "{group}\Uninstall PheebKit"; Filename: "{uninstallexe}"
Name: "{autodesktop}\菲比工具箱 (PheebKit)"; Filename: "{app}\PheebKit.exe"; Tasks: desktopicon

[Run]
Filename: "{app}\PheebKit.exe"; Description: "{cm:LaunchProgram,PheebKit}"; Flags: nowait postinstall skipifsilent; Check: HasWebView2
Filename: "https://developer.microsoft.com/en-us/microsoft-edge/webview2/#download-section"; Description: "{cm:GetWebView2}"; Flags: shellexec postinstall skipifsilent unchecked; Check: NeedsWebView2

[CustomMessages]
english.GetWebView2=Get Microsoft WebView2 Runtime
chinesesimplified.GetWebView2=获取 Microsoft WebView2 运行时
english.UninstallBlocked=Please close PheebKit before uninstalling. Active media jobs will not be interrupted.
chinesesimplified.UninstallBlocked=请先关闭菲比工具箱，再进行卸载。正在处理的媒体任务不会被强制中断。
english.WebView2Missing=PheebKit is installed. Install Microsoft WebView2 Runtime from the official download page before launching. FFmpeg can be configured inside PheebKit.
chinesesimplified.WebView2Missing=菲比工具箱已安装。启动前请从微软官方下载页面安装 WebView2 运行时。FFmpeg 可在 菲比工具箱设置中补全。

[Code]
function InitializeUninstall: Boolean;
begin
  Result := not CheckForMutexes('Local\kompeg-application-running');
  if not Result then
    SuppressibleMsgBox(ExpandConstant('{cm:UninstallBlocked}'), mbError, MB_OK, IDOK);
end;

function RuntimeAt(Root: Integer; Subkey: String): Boolean;
var Version: String;
begin
  Result := RegQueryStringValue(Root, Subkey, 'pv', Version) and
    (Version <> '') and (Version <> '0.0.0.0');
end;

function HasWebView2: Boolean;
var Key: String;
begin
  Key := 'Software\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}';
  Result := RuntimeAt(HKCU, Key) or RuntimeAt(HKLM32, Key) or RuntimeAt(HKLM64, Key);
end;

function NeedsWebView2: Boolean;
begin
  Result := not HasWebView2;
end;

procedure CurPageChanged(CurPageID: Integer);
begin
  if (CurPageID = wpFinished) and NeedsWebView2 then
    WizardForm.FinishedLabel.Caption := ExpandConstant('{cm:WebView2Missing}');
end;
