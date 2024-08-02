all:
	go build -o cmd/cli/cli cmd/cli/main.go

test:
	go test -v .