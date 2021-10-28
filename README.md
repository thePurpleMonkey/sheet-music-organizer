# Sheet Music Organizer

A web app to categorize and organize sheet music.
A live version of this website can be found at https://sheetmusicorganizer.com.

## Features

- Manage multiple collections of sheet music
- Organize songs with tags
- Collaborate with other users
- Plan and share performances with setlists
- Search for songs with a variety of filters
- Responsive design for mobile

## Technologies

This project was built with:
 - [Go](golang.org)
 - HTML/CSS/JavaScript
 - [Bootstrap 4](getbootstrap.com)
 - [PostgreSQL](postgresql.org)
 - [SendGrid](sendgrid.com)
 
The build pipeline is managed through shell scripts. Batch on Windows, Bash on Linux.
This project was tested with Go versions 1.10.4, 1.16, and 1.17 and PostgreSQL version 10.18 on Linux and 11.4 on Windows.

## Building and Running

The default build files for this project are designed for development on Windows,
and deploying to a production web server on Linux. 

### Database

This project was developed using a PostgreSQL database backend,
but any SQL database should do with some modifications.

1. Create a new database and user with full permissions to the database.
2. Run the `sql/create_database.sql` script to create all the necessary tables and stored procedures.
   - The stored procedures `search_collection` and `advanced_search_collection` depend on features provided by PostgreSQL.
   If a different SQL backend is used, then they will likely need to be modified.

### Windows

The `build.bat` file for Windows sets environment variables, builds the executable, and runs the server.
First edit `build.bat` and set the environment variables, 
then run `build.bat` to build and start the server.

### Linux

Linux has three stages to build and run the executable.
The `set-env.sh` script sets the environment variables, `build.sh` builds the executable, and `run.sh` runs it. 
1. First edit `set-env.sh` to set the environment variables to the correct values.
2. Execute `build.sh` to build the executable. If there is no output, the build was successful.
3. Execute `run.sh` to start the server.

If an instance of the server is already running, any changes to the HTML, CSS, or JavaScript will take effect immediately.
However, any changes to the Go source code will have to be built first and
the server restarted before the changes are live.
You can take advantage of this to download and build a new version of the code while the old version sill runs in the background.
Just restart the server to run the updated version.

### Environment Variables
The following environment variables are required to be set for the application to run.
These environment variables are read by the server on demand, so changing them
while the server is running will cause it to read the new values immediately.

| Name | Description | Example |
| ---- | ----------- | ------- |
| SESSION_KEY | Key used to encrypt session and cookie data. | `$(cat session_key.txt)` |
| SENDGRID_API_KEY | API key for your SendGrid account. Used to send emails with the SendGrid API. | `$(cat sendgrid_api_key.txt)` |
| HOST | This is the FQDN that the website is running under. This value is used in emails as the host part of the URL. | `example.com` |
| CERT_FILE | Path to the certificate file for the https server. | `~/certs/localhost.crt` |
| KEY_FILE | Path to the key file for the https server. | `~/certs/localhost.key` |
| PORT | Port number to run the server on. | `8000` |
| DB_USERNAME | Username to log into the database with. | `smo` |
| DB_PASSWORD | Password to log into the database with. | `$(cat db_password.txt)` |
| ADMIN_EMAIL | Email address to include in emails and various other places on the website. | `admin@example.com` |
| LOG_PATH | The directory to store the log file in. | `/var/log/` |