@echo off

REM ============= build script for windows ================
REM how to use
REM win_build.bat v0.0.1
REM =======================================================

REM ============= variable definitions ================
set currentDir=%CD%
set output=out
set name=aliyunpan
set version=%1

REM ============= build action ================
call :build_task %name%-%version%-windows-x86 windows 386
call :build_task %name%-%version%-windows-x64 windows amd64
call :build_task %name%-%version%-linux-386 linux 386
call :build_task %name%-%version%-linux-amd64 linux amd64
call :build_task %name%-%version%-darwin-macos-amd64 darwin amd64

goto:EOF

REM ============= build function ================
:build_task
setlocal

set targetName=%1
set GOOS=%2
set GOARCH=%3
set goarm=%4
set GO386=sse2
set CGO_ENABLED=0
set GOARM=%goarm%

echo "Building %targetName% ..."
if %GOOS% == windows (
  goversioninfo -o=resource_windows_386.syso
  goversioninfo -64 -o=resource_windows_amd64.syso
  go build -ldflags "-linkmode internal -X main.Version=%version% -s -w" -o "%output%/%1/%name%.exe"
) ^
else (
  go build -ldflags "-X main.Version=%version% -s -w" -o "%output%/%1/%name%"
)

copy README.md %output%\%1

mkdir %output%\%1\plugin
xcopy /e assets\plugin %output%\%1\plugin

mkdir %output%\%1\sync_drive
xcopy /e assets\sync_drive %output%\%1\sync_drive
endlocal

