FROM golang:1.26-bookworm AS builder
WORKDIR /src
ENV CGO_ENABLED=0 GOFLAGS=-mod=readonly
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -trimpath -ldflags="-s -w" -o /out/app ./cmd/app

# debug-nonroot includes busybox (/busybox/wget) for Docker healthchecks without app changes
FROM gcr.io/distroless/static-debian12:debug-nonroot AS runtime
WORKDIR /app
COPY --from=builder /out/app /app/app
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/app/app"]

FROM runtime AS final
