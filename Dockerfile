# Multi-stage build for EasyPlan (Konzept.md, section 8).
#
# Build stage: Go 1.27.3, static binary without CGO. The minimum version
# is declared in go.mod; this image may move to newer patch releases of the
# same minor version (1.27.3 -> 1.27.4) through Dependabot, because the build
# stays valid with the newer patch version. A minor bump (1.27 -> 1.28)
# changes the go directive in go.mod and is done manually in a commit that
# changes go.mod and this Dockerfile together.
# Runtime stage: distroless, without shell and without root. Only /data is
# written, everything else stays read-only.
#
# Both base images are pinned to a digest. Dependabot keeps tag and digest in
# sync for the "docker" ecosystem, so the pin stays verifiable over time.

# --- Build stage -------------------------------------------------------------
FROM golang:1.27.3-bookworm@sha256:d9b6a1beb9474adc2a1ce842573639b44b2a0dd49b6669f51d3821baf9ec37c9 AS build

WORKDIR /src

# Dependencies first, so this layer stays in the cache when the code changes.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Static binary: no CGO, no paths from the build machine, no debug symbols.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-s -w" -o /out/easyplan ./cmd/easyplan

# Create the data directory and hand it over to the runtime user. The
# distroless stage has no shell, so this happens here.
RUN mkdir -p /out/data && chown 65532:65532 /out/data

# --- Runtime stage -----------------------------------------------------------
FROM gcr.io/distroless/static-debian12:nonroot@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab

COPY --from=build /out/easyplan /easyplan
# /data belongs to the nonroot user, so that a named volume is writable.
COPY --from=build --chown=65532:65532 /out/data /data

ENV EASYPLAN_ADDR=:8080 \
    EASYPLAN_DATA=/data \
    TZ=Europe/Zurich

USER 65532:65532

VOLUME /data
EXPOSE 8080

# The healthcheck calls the subcommand of the binary itself: no shell, no curl.
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD ["/easyplan", "healthcheck"]

ENTRYPOINT ["/easyplan"]
