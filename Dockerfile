FROM golang:1.25.3 AS builder

WORKDIR /app
COPY . .
RUN go build -o pipeline-generator .

FROM golang:1.25.3

COPY --from=builder /app/pipeline-generator /usr/local/bin/pipeline-generator

ENTRYPOINT [ "pipeline-generator" ]
