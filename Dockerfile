# syntax=docker/dockerfile:1.7

# ---- Build stage ----
# golang:alpine is small and supports all our target architectures natively.
# BuildKit sets TARGETOS and TARGETARCH automatically based on --platform.
FROM --platform=$BUILDPLATFORM golang:1.23-alpine AS build

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev

WORKDIR /src

# Cache the module download layer separately from the source.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build a fully static binary, stripped of symbol table and DWARF info,
# with the version baked in. CGO_ENABLED=0 is non-negotiable.
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build \
    -trimpath \
    -ldflags="-s -w -X main.version=$VERSION" \
    -o /out/lpaas98 \
    ./cmd/lpaas98

# ---- Runtime stage ----
# scratch is empty: no shell, no package manager, no libc. Just our binary.
# The binary is fully static so it needs nothing else.
FROM scratch

# Copy CA certificates from the build stage so HTTPS catalog fetches work.
# (The only network egress the server does is fetching catalog.json.)
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY --from=build /out/lpaas98 /lpaas98

# Volumes for host-managed state. Both directories are read on startup
# and the installer writes to /app/games.
VOLUME ["/app/games", "/app/assets"]
WORKDIR /app

EXPOSE 9898

ENTRYPOINT ["/lpaas98"]
CMD ["server", "--addr", "0.0.0.0:9898", "--games-dir", "/app/games", "--assets-dir", "/app/assets"]
