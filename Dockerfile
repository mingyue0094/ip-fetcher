FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY . .

RUN go build -o ip-fetcher main.go

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/ip-fetcher .

CMD ["./ip-fetcher"]
