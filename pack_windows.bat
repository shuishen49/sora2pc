@echo off
setlocal

set "PROJECT_DIR=%~dp0"
set "OUT_DIR=%PROJECT_DIR%dist\sorapc-win"
set "EXE_SRC="

pushd "%PROJECT_DIR%"
wails build
if errorlevel 1 (
  echo Build failed.
  popd
  exit /b 1
)
popd

if exist "%PROJECT_DIR%build\bin\sorapc.exe" set "EXE_SRC=%PROJECT_DIR%build\bin\sorapc.exe"
if not defined EXE_SRC if exist "%PROJECT_DIR%sorapc.exe" set "EXE_SRC=%PROJECT_DIR%sorapc.exe"

if not defined EXE_SRC (
  echo EXE not found. Run: wails build at least once.
  exit /b 1
)

if exist "%OUT_DIR%" rmdir /s /q "%OUT_DIR%"
mkdir "%OUT_DIR%"

copy /y "%EXE_SRC%" "%OUT_DIR%\sorapc.exe" >nul

if exist "%PROJECT_DIR%config.json" copy /y "%PROJECT_DIR%config.json" "%OUT_DIR%\config.json" >nul
if exist "%PROJECT_DIR%logs.txt" copy /y "%PROJECT_DIR%logs.txt" "%OUT_DIR%\logs.txt" >nul

(
  echo @echo off
  echo cd /d "%%~dp0"
  echo start "" "%%~dp0sorapc.exe"
) > "%OUT_DIR%\run_sorapc.bat"

echo Pack done: %OUT_DIR%
