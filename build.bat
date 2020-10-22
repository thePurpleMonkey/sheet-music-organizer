@echo off

echo Building...
set /P SESSION_KEY=<key.txt
set /P SENDGRID_API_KEY=<SendGrid_API_key.txt
go build -ldflags="-linkmode=internal -extld=none"
if /I "%ERRORLEVEL%" NEQ "0" (
	echo Build failed.
) ELSE (
	echo Running...
	sheet-music-organizer.exe
)