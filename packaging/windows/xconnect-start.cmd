@echo off
REM 登录后自动启动：后台运行 xconnect -daemon -sync，再启动托盘
set INSTALLDIR=%~dp0
start "" /B "%INSTALLDIR%xconnect.exe" -daemon -sync
start "" "%INSTALLDIR%xconnect-tray.exe"
