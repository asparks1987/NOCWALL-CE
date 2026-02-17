# API builder image (BuildKit-independent)
FROM golang:1.21-alpine AS api-builder
WORKDIR /src
ENV GOSUMDB=off

COPY api/go.mod api/go.sum ./
RUN go mod download

COPY api/ ./
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} go build -o /out/api .

# SPA preview image
FROM node:20-alpine AS spa-builder
WORKDIR /app
COPY web/ ./

# Final image
FROM alpine:3.19
WORKDIR /app
COPY --from=api-builder /out/api /app/api
COPY --from=spa-builder /app /app/web
ENV API_ADDR=:8080
EXPOSE 8080
CMD ["/app/api"]
