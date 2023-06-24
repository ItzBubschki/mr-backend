FROM golang:alpine AS builder

WORKDIR /main
COPY . .

RUN go build -o /exec-be main/main.go

FROM alpine:latest
COPY --from=builder /exec-be /mr-be
ENTRYPOINT ["/mr-be"]