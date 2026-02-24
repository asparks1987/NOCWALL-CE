# Base images are configurable so builds can switch registries when Docker Hub DNS is unavailable.
ARG GO_BASE_IMAGE=golang:1.21-alpine
ARG NODE_BASE_IMAGE=node:20-alpine
ARG ALPINE_BASE_IMAGE=alpine:3.19

# API builder image (BuildKit-independent)
FROM ${GO_BASE_IMAGE} AS api-builder
WORKDIR /src
ENV GOSUMDB=off

COPY api/go.mod api/go.sum ./
RUN go mod download

COPY api/ ./
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} go build -o /out/api .

# SPA preview image
FROM ${NODE_BASE_IMAGE} AS spa-builder
WORKDIR /app
COPY web/ ./

# Final image
FROM ${ALPINE_BASE_IMAGE}
WORKDIR /app
COPY --from=api-builder /out/api /app/api
COPY --from=spa-builder /app /app/web
ENV API_ADDR=:8080
EXPOSE 8080
CMD ["/app/api"]
