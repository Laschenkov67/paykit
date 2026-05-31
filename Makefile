.PHONY: test lint cover doc tidy

test:
	go test -race ./...

lint:
	golangci-lint run

cover:
	go test -race -coverprofile=cover.out ./...
	go tool cover -html=cover.out

doc:
	godoc -http=:6060

tidy:
	go mod tidy