FROM golang:1.22-alpine AS build_base
RUN apk add --no-cache git gcc ca-certificates libc-dev
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY ./ ./
RUN CGO_ENABLED=0 go build -ldflags "-w -s" -trimpath -o speedtest .

FROM gcr.io/distroless/static-debian12

USER nobody

WORKDIR /app
COPY --from=build_base /build/speedtest ./
COPY settings.toml ./

EXPOSE 8989

CMD ["./speedtest"]
