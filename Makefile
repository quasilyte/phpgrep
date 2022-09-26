.PHONY: test lint

GOPATH_DIR=`go env GOPATH`

export GO111MODULE := on

test:
	go test -v -race -count=1 -coverprofile=coverage.out ./...

lint:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOPATH_DIR)/bin v1.30.0
	$(GOPATH_DIR)/bin/golangci-lint run -v
