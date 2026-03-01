FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod .
RUN go mod download
COPY . .
RUN go build -ldflags="-s -w" -o aether-bootstrap .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/aether-bootstrap .
EXPOSE 7070
CMD ["./aether-bootstrap"]
