. ./set-env.sh

go build -ldflags="-linkmode=internal -extld=none"
if [ $? -ne 0 ]; then
	echo Build failed.
# else
# 	echo Starting server...
# 	sudo --preserve-env ./sheet-music-organizer &>> /home/michael/sheet-music-organizer.log &
fi
