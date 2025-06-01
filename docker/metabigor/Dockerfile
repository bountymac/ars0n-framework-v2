FROM golang:1.20-alpine

ENV ASSUME_NO_MOVING_GC_UNSAFE_RISK_IT_WITH=go1.20

RUN go install github.com/j3ssie/metabigor@latest

ENTRYPOINT ["/go/bin/metabigor"] 