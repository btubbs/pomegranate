all: bin/pomegranate

bin/pomegranate: cmd/main.go
	go build -o $@ $<
