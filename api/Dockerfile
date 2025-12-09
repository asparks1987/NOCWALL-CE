# API builder image
FROM golang:1.21-alpine AS api-builder
WORKDIR /src
COPY api/go.mod ./
RUN go mod download
COPY api/ ./
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/api ./main.go

# SPA preview image
FROM node:20-alpine AS spa-builder
WORKDIR /app
COPY web/ ./
# No build step; static assets only

# Final image
FROM alpine:3.19
WORKDIR /app
COPY --from=api-builder /out/api /app/api
COPY --from=spa-builder /app /app/web
ENV API_ADDR=:8080
EXPOSE 8080
CMD ["/app/api"]
