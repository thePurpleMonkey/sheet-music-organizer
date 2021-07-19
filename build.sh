. ./set-env.sh

go build -ldflags="-linkmode=internal -extld=none"
if [ $? -ne 0 ]; then
	echo Build failed.
fi
