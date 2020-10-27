@echo off

echo Building...
set /P SESSION_KEY=<key.txt
set /P SENDGRID_API_KEY=<SendGrid_API_key.txt
set HOST=localhost:8000
set CERT_PATH=C:\Users\mhump\.ssh
set CERT_BASE_NAME=localhost
set PORT=8000
set DB_USERNAME=smo
set DB_PASSWORD=smo-test
go build -ldflags="-linkmode=internal -extld=none"
if /I "%ERRORLEVEL%" NEQ "0" (
	echo Build failed.
) ELSE (
	echo Running...
	sheet-music-organizer.exe
)