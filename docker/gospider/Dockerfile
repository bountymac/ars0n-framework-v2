FROM golang:1.23-alpine

RUN GO111MODULE=on go install github.com/jaeles-project/gospider@latest

WORKDIR /app

ENTRYPOINT ["/go/bin/gospider"] 