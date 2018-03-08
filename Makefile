.PHONY: tests viewcoverage

all: tests 

bin/pmg: cmd/pmg.go
	go build -o $@ $<

tests:
	go test .

viewcoverage:
	go test -coverprofile=coverage.out
	go tool cover -html=coverage.out
