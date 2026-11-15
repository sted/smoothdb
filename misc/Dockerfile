# Build from the repository root:
#   docker build -f misc/Dockerfile --build-arg VERSION=$(git describe --tags --always) -t smoothdb .
# Run:
#   docker run -p 4000:4000 -e SMOOTHDB_DATABASE_URL=... -e SMOOTHDB_JWT_SECRET=... smoothdb
#
# The image holds only the binary. The configuration comes from the
# SMOOTHDB_* environment variables or from /data/config.jsonc on a volume.
# The server writes config.jsonc and smoothdb.log in /data, as user nonroot.
# The build has no cgo, like the release binaries, so Go plugins cannot load.

FROM golang:1.27.4-alpine AS builder
ARG VERSION=dev
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath \
        -ldflags "-s -w -X github.com/sted/smoothdb/version.Version=${VERSION}" \
        -o /out/smoothdb . \
    && mkdir -p /out/data

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/smoothdb /smoothdb
COPY --from=builder --chown=nonroot:nonroot /out/data /data
WORKDIR /data
USER nonroot
EXPOSE 4000
ENTRYPOINT ["/smoothdb"]
