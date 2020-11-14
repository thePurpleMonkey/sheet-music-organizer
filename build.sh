#export SESSION_KEY=$(cat session_key.txt)
#export SENDGRID_API_KEY=$(cat sendgrid_api_key.txt)
#export HOST=smo.michaelhumphrey.dev
#export CERT_FILE=/etc/letsencrypt/live/michaelhumphrey.dev/fullchain.pem
#export KEY_FILE=/etc/letsencrypt/live/michaelhumphrey.dev/privkey.pem
#export PORT=8000
#export DB_USERNAME=smo_user
#export DB_PASSWORD=$(cat db_password.txt)

. ./set-env.sh

go build -ldflags="-linkmode=internal -extld=none"
if [ $? -ne 0 ]; then
	echo Build failed.
else
	echo Starting server...
	sudo --preserve-env ./sheet-music-organizer &>> /home/michael/sheet-music-organizer.log &
fi
