# syntax=docker/dockerfile:1

# Build on the machine's own architecture and cross-compile for the target, so multi-platform
# builds need no emulation.
FROM --platform=$BUILDPLATFORM golang:1.23-alpine AS build
ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /out/design-export-docs .

# The binary is static and makes no network calls, so nothing else is needed: no shell, no
# certificates, no packages.
FROM scratch
ARG VERSION=dev
LABEL org.opencontainers.image.title="design-export-docs" \
      org.opencontainers.image.description="Documentation sites from exported design-system folders" \
      org.opencontainers.image.source="https://github.com/joshmcarthur/design-export-docs" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.version="${VERSION}"
COPY --from=build /out/design-export-docs /design-export-docs
USER 65532:65532
WORKDIR /work
ENTRYPOINT ["/design-export-docs"]
