# Build the spiffe-helper binary
ARG go_version
FROM --platform=$BUILDPLATFORM golang:${go_version}-alpine AS base
WORKDIR /workspace

# Cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
COPY go.* ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

# Copy the go source
COPY cmd/spiffe-helper/ cmd/spiffe-helper/
COPY pkg/ pkg/

# xx is a helper for cross-compilation
# when bumping to a new version analyze the new version for security issues
# then use crane to lookup the digest of that version so we are immutable
# crane digest tonistiigi/xx:1.3.0
FROM --platform=${BUILDPLATFORM} tonistiigi/xx@sha256:904fe94f236d36d65aeb5a2462f88f2c537b8360475f6342e7599194f291fb7e AS xx

FROM --platform=${BUILDPLATFORM} base AS builder
ARG TARGETPLATFORM
ARG TARGETARCH

ENV CGO_ENABLED=0
COPY --link --from=xx / /
RUN xx-go --wrap
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    go build -o bin/spiffe-helper cmd/spiffe-helper/main.go

WORKDIR /
RUN ln -s /workspace/bin/spiffe-helper /spiffe-helper
RUN go install github.com/go-delve/delve/cmd/dlv@latest
EXPOSE 40000
ENTRYPOINT ["/go/bin/dlv", "--listen=:4000", "--headless=true", "--accept-multiclient", "--api-version=2", "exec", "/spiffe-helper", "--"]
CMD []
