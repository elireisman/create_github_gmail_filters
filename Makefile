all: keygen
	go run -v create_filters.go

local: keygen
	go run -v create_filters.go -local=true

keygen:
	if ! which -s openssl ; then echo "Run `brew install openssl` and retry" ; exit 1 ; fi
	if [ ! -e server.key ] ; then openssl genrsa -out server.key 2048 ; openssl req -new -x509 -sha256 -key server.key -out server.crt -days 30 -subj "/C=US/ST=Washington/L=Kirkland/O=Bob/OU=ImportantRole/CN=localhost" ; fi

clean:
	rm -f $(HOME)/.credentials/gmail-go-quickstart.json
	rm -f create_filters
	rm -f server.*

