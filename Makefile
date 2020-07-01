.PHONY: tests viewcoverage check ci

GOBIN ?= $(GOPATH)/bin

all: tests check

bin/pmg: pmg/pmg.go
	go build -o $@ $<

tests:
	go test . -mod=readonly

profile.cov:
	go test -coverprofile=$@ -mod=readonly

viewcoverage: profile.cov 
	go tool cover -html=$<

check: $(GOBIN)/golangci-lint
	$(GOBIN)/golangci-lint run

$(GOBIN)/goveralls:
	go get -v -u github.com/mattn/goveralls

$(GOBIN)/golangci-lint:
	curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s -- -b $(GOPATH)/bin v1.12.3

ci: profile.cov check $(GOBIN)/goveralls
	$(GOBIN)/goveralls -coverprofile=$< -service=travis-ci
