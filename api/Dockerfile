# API builder image
# syntax=docker/dockerfile:1.4
FROM golang:1.21-alpine AS api-builder
WORKDIR /src
ENV GOSUMDB=off
COPY api/go.mod ./
COPY api/ ./
# Cache modules and build cache to speed up multi-arch builds
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download && go mod tidy
ARG TARGETOS
ARG TARGETARCH
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} go build -o /out/api .

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
