@echo off

echo Building...
set /P SESSION_KEY=
set /P SENDGRID_API_KEY=
set HOST=localhost:8000
set CERT_FILE=
set KEY_FILE=
set PORT=8000
set DB_USERNAME=
set DB_PASSWORD=
set ADMIN_EMAIL=
go build -ldflags="-linkmode=internal -extld=none"
if /I "%ERRORLEVEL%" NEQ "0" (
	echo Build failed.
) ELSE (
	echo Running...
	sheet-music-organizer.exe
)