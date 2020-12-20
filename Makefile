linter:
	go vet ./...
	golangci-lint run -E gofmt -E golint -E vet --timeout 2m

test:
	go test -v --race ./...

doc-gen:
	swag init --generalInfo=./cmd/main.go --parseInternal

run:
	docker-compose up
