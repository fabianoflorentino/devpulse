# ─────────────────────────────────────────────────────────────────────────────
# Stage 1 – dependency cache
#   Separate layer so `go mod download` is only re-run when go.mod/go.sum
#   change, keeping rebuild times fast.
# ─────────────────────────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS deps

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

# ─────────────────────────────────────────────────────────────────────────────
# Stage 2 – build
#   CGO_ENABLED=0  → fully static binary (required for distroless)
#   -trimpath      → removes local build paths from the binary
#   -ldflags       → strips debug info, embeds version from build arg
# ─────────────────────────────────────────────────────────────────────────────
FROM deps AS builder

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

WORKDIR /src

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
      -trimpath \
      -ldflags="-s -w \
        -X main.version=${VERSION} \
        -X main.commit=${COMMIT} \
        -X main.buildDate=${BUILD_DATE}" \
      -o /out/devpulse \
      .

# ─────────────────────────────────────────────────────────────────────────────
# Stage 3 – development
#   Full Alpine image with shell, git and Air (live-reload) available.
#   Mounts the source tree via docker-compose volumes for hot-reload.
# ─────────────────────────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS dev

RUN apk add --no-cache git ca-certificates tzdata && \
    go install github.com/air-verse/air@latest

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

# Source is provided at runtime via bind-mount (see docker-compose.yaml).
# Air watches for changes and rebuilds automatically.
COPY .air.toml ./

CMD ["air"]

# ─────────────────────────────────────────────────────────────────────────────
# Stage 4 – production (distroless)
#   gcr.io/distroless/static-debian12 contains only:
#     • CA certificates
#     • tzdata
#     • /etc/passwd (nobody user)
#   No shell, no package manager, minimal attack surface.
# ─────────────────────────────────────────────────────────────────────────────
FROM gcr.io/distroless/static-debian12:nonroot AS production

# Copy the statically linked binary from the builder stage.
COPY --from=builder /out/devpulse /devpulse

# Run as the non-root "nobody" user provided by distroless/nonroot.
USER nonroot:nonroot

ENTRYPOINT ["/devpulse"]
