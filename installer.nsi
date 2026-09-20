; The Windows installer, built from Linux by `makensis` (no Wine, no Windows
; box, free/open source). Replaces the paid Hydraulic Conveyor MSIX path.
;
; Per-user install: no admin rights, no UAC prompt, matches how the loose exe
; already ran. The app's own in-app updater (update.go) is what keeps it
; current after this; this installer's only job is first install: put the
; exe somewhere permanent, add a Start Menu entry, and give it a real
; uninstaller in Add/Remove Programs.
;
; Invoked by scripts/package.sh as:
;   makensis -DVERSION=<version> -DSRC=<path to signed CLIque.exe> -DOUT=<output path> installer.nsi

!ifndef VERSION
  !define VERSION "0.0.0"
!endif
!ifndef SRC
  !define SRC "dist\CLIque.exe"
!endif
!ifndef OUT
  !define OUT "dist\CLIque-Setup.exe"
!endif

!include "MUI2.nsh"

Name "CLIque"
OutFile "${OUT}"
Unicode true
InstallDir "$LOCALAPPDATA\Programs\CLIque"
RequestExecutionLevel user
VIProductVersion "${VERSION}.0"
VIAddVersionKey "ProductName" "CLIque"
VIAddVersionKey "CompanyName" "Willhite Strategy Group"
VIAddVersionKey "FileVersion" "${VERSION}"
VIAddVersionKey "ProductVersion" "${VERSION}"
VIAddVersionKey "FileDescription" "CLIque installer"
VIAddVersionKey "LegalCopyright" "MIT licensed"
!define MUI_ICON "tray.ico"
!define MUI_UNICON "tray.ico"

!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "English"

!define UNINST_KEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\CLIque"

Section "Install"
  SetOutPath "$INSTDIR"
  File "${SRC}"
  File "tray.ico"

  CreateDirectory "$SMPROGRAMS\CLIque"
  CreateShortcut "$SMPROGRAMS\CLIque\CLIque.lnk" "$INSTDIR\CLIque.exe" "" "$INSTDIR\tray.ico"
  CreateShortcut "$SMPROGRAMS\CLIque\Uninstall CLIque.lnk" "$INSTDIR\Uninstall.exe"

  WriteUninstaller "$INSTDIR\Uninstall.exe"

  ; Per-user (HKCU), so install and uninstall both need no elevation.
  WriteRegStr HKCU "${UNINST_KEY}" "DisplayName" "CLIque"
  WriteRegStr HKCU "${UNINST_KEY}" "DisplayVersion" "${VERSION}"
  WriteRegStr HKCU "${UNINST_KEY}" "Publisher" "Willhite Strategy Group"
  WriteRegStr HKCU "${UNINST_KEY}" "DisplayIcon" "$INSTDIR\tray.ico"
  WriteRegStr HKCU "${UNINST_KEY}" "InstallLocation" "$INSTDIR"
  WriteRegStr HKCU "${UNINST_KEY}" "UninstallString" '"$INSTDIR\Uninstall.exe"'
  WriteRegDWORD HKCU "${UNINST_KEY}" "NoModify" 1
  WriteRegDWORD HKCU "${UNINST_KEY}" "NoRepair" 1

  Exec '"$INSTDIR\CLIque.exe"'
SectionEnd

Section "Uninstall"
  ; The updater leaves .old/.new/.new.sha256 beside the exe between restarts;
  ; a straight wildcard clears whichever of those happen to exist.
  Delete "$INSTDIR\CLIque.exe*"
  Delete "$INSTDIR\tray.ico"
  Delete "$INSTDIR\Uninstall.exe"
  RMDir "$INSTDIR"

  Delete "$SMPROGRAMS\CLIque\CLIque.lnk"
  Delete "$SMPROGRAMS\CLIque\Uninstall CLIque.lnk"
  RMDir "$SMPROGRAMS\CLIque"

  DeleteRegKey HKCU "${UNINST_KEY}"
SectionEnd
