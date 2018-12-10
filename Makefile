.PHONY: tests viewcoverage check dep ci

GOBIN ?= $(GOPATH)/bin

all: tests check

bin/pmg: pmg/pmg.go
	go build -o $@ $<

dep: $(GOBIN)/dep
	$(GOBIN)/dep ensure -v

tests: dep
	go test .

profile.cov: dep
	go test -coverprofile=$@

viewcoverage: profile.cov 
	go tool cover -html=$<

vet:
	go vet ./...

check: $(GOBIN)/golangci-lint
	$(GOBIN)/golangci-lint run

$(GOBIN)/goveralls:
	go get -v -u github.com/mattn/goveralls

$(GOBIN)/dep:
	go get -v -u github.com/golang/dep/cmd/dep

$(GOBIN)/golangci-lint:
	go get -v -u github.com/golangci/golangci-lint
	cd $GOPATH/src/github.com/golangci/golangci-lint/cmd/golangci-lint
	go install -ldflags "-X 'main.version=$(git describe --tags)' -X 'main.commit=$(git rev-parse --short HEAD)' -X 'main.date=$(date)'"	

ci: profile.cov vet check $(GOBIN)/goveralls
	$(GOBIN)/goveralls -coverprofile=$< -service=travis-ci
