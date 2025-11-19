FROM golang:1.25.3

WORKDIR /app

COPY . .

ENTRYPOINT [ "go", "run", "." ]
