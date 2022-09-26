VERSION = $(shell git describe --tags | head -1)

mackerel-plugin-sendgrid: go.mod go.sum *.go
	go build -ldflags "-s -w -X main.version=${VERSION}" -trimpath
