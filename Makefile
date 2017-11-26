all:
	go run -v create_filters.go

local:
	go run -v create_filters.go -local=true

clean:
	rm -f $(HOME)/.credentials/gmail-go-quickstart.json
	rm -f ./create_filters

