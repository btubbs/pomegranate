.PHONY: tests

all: bin/pomegranate

bin/pomegranate: cmd/main.go
	go build -o $@ $<

tests:
	go test -run ''
