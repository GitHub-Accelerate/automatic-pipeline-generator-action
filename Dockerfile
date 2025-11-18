FROM golang:1.25.2

ENTRYPOINT [ "go run main.go arg0 arg1 arg2" ]
