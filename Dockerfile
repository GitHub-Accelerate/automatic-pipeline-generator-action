FROM golang:1.25.3

ENTRYPOINT [ "go", "run", "*.go" ]
