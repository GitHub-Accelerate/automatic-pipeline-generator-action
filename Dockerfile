FROM golang:1.25.3

ENTRYPOINT [ "go", "run", "main.go", "ordered_map.go", "git.go" ]
