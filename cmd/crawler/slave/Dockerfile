# Build from the repository root:
#   docker build -f cmd/crawler/slave/Dockerfile -t prawler-slave .

FROM golang:1.25.6-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/slave ./cmd/crawler/slave

# distroless/static ships CA certificates, needed for fetching HTTPS pages
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app

COPY --from=build /out/slave ./slave
COPY config.json ./

ENTRYPOINT ["./slave"]
