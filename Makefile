.PHONY: tests viewcoverage

all: bin/pomegranate

bin/pomegranate: cmd/main.go
	go build -o $@ $<

tests:
	go test .

viewcoverage:
	go test -coverprofile=coverage.out
	go tool cover -html=coverage.out
