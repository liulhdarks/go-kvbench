all:
	go build -o kvbench cmd/kvbench/main.go \
    go build -o cli cmd/cli/main.go

test:
	go test -v .