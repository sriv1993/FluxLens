# Multi-stage build for all FluxLens Go services. Build args select which
# command to compile so a single Dockerfile produces every service image.
#
#   docker build --build-arg CMD=curator -t fluxlens/curator:dev .

ARG GO_VERSION=1.22

FROM golang:${GO_VERSION}-alpine AS build
ARG CMD
WORKDIR /src

RUN apk add --no-cache git ca-certificates && update-ca-certificates

COPY go.mod go.sum* ./
RUN go mod download

COPY . .

RUN test -n "${CMD}" || (echo "CMD build-arg is required (curator|api-gateway|orchestrator|audit-writer|ingest-mysql|synth-source)" && exit 1)
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-s -w" -o /out/fluxlens-${CMD} ./cmd/${CMD}

FROM gcr.io/distroless/static:nonroot
ARG CMD
COPY --from=build /out/fluxlens-${CMD} /usr/local/bin/fluxlens
USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/fluxlens"]
