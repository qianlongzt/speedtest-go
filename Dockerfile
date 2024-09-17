FROM --platform=$BUILDPLATFORM golang:1.23.1-alpine AS build_base
ARG BUILDKIT_SBOM_SCAN_STAGE=true
ARG TARGETOS TARGETARCH

RUN apk add --no-cache git gcc ca-certificates libc-dev
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY ./ ./
RUN GOOS=$TARGETOS GOARCH=$TARGETARCH CGO_ENABLED=0 \
  go build -ldflags "-w -s" -trimpath -o speedtest .

FROM scratch

WORKDIR /app
COPY --from=build_base /build/speedtest ./
COPY settings.toml ./

EXPOSE 8989

CMD ["./speedtest"]
