!include "MUI2.nsh"

!define PRODUCT_NAME "BaudLink"
!define PRODUCT_VERSION "1.0.0"
!define PRODUCT_PUBLISHER "BaudLink"
!define PRODUCT_EXE_NAME "baudlink-service.exe"

Name "${PRODUCT_NAME} ${PRODUCT_VERSION}"
OutFile "${PRODUCT_NAME}-installer.exe"
InstallDir "$PROGRAMFILES\\${PRODUCT_NAME}"
RequestExecutionLevel admin

Page directory
Page instfiles

Section "Install"
    SetOutPath "$INSTDIR"
    File "..\\build\\baudlink-service.exe"
    File "..\\build\\baudlink-cli.exe"

    ; Create uninstall information
    WriteUninstaller "$INSTDIR\\uninstall.exe"

    ; Optionally register Windows service via sc.exe
    ; Using nsExec to run sc create (service name BaudLink)
    ; If service exists, attempt to delete first
    nsExec::ExecToLog 'sc query "BaudLink"'
    Pop $0
    StrCmp $0 0 +2
    nsExec::ExecToLog 'sc delete "BaudLink"'

    nsExec::ExecToLog 'sc create "BaudLink" binPath= "$INSTDIR\\baudlink-service.exe" start= auto'
    nsExec::ExecToLog 'sc description "BaudLink" "Serial port background service"'

SectionEnd

Section "Uninstall"
    nsExec::ExecToLog 'sc stop "BaudLink"'
    nsExec::ExecToLog 'sc delete "BaudLink"'
    Delete "$INSTDIR\\baudlink-service.exe"
    Delete "$INSTDIR\\baudlink-cli.exe"
    Delete "$INSTDIR\\uninstall.exe"
    RMDir "$INSTDIR"
SectionEnd
