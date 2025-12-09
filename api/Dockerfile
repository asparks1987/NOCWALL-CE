FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o server ./main.go

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/server /app/server
ENV API_ADDR=:8080
EXPOSE 8080
ENTRYPOINT ["/app/server"]
