.PHONY: tests viewcoverage check dep ci

GOLIST=$(shell go list ./...)
GOBIN ?= $(GOPATH)/bin

all: tests check

bin/pmg: pmg/pmg.go
	go build -o $@ $<

dep: $(GOBIN)/dep
	$(GOBIN)/dep ensure

tests: dep
	go test .

viewcoverage: dep
	go test -coverprofile=coverage.out
	go tool cover -html=coverage.out

check: $(GOBIN)/megacheck
	go vet $(GOLIST)
	$(GOBIN)/megacheck $(GOLIST)

$(GOBIN)/megacheck:
	go get -v -u honnef.co/go/tools/cmd/megacheck

$(GOBIN)/goveralls:
	go get -v -u github.com/mattn/goveralls

$(GOBIN)/dep:
	go get -v -u github.com/golang/dep/cmd/dep

ci: dep check $(GOBIN)/goveralls
	goveralls -coverprofile=profile.cov -service=travis-ci
