FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod ./
COPY main.go ./
RUN go build -o contact-server .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/contact-server .
EXPOSE 1337
CMD ["./contact-server"]
