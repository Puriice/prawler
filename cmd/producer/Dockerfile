# Build from the repository root:
#   docker build -f cmd/producer/Dockerfile -t prawler-producer .
# seeds.json is not baked in; mount it at /app/seeds.json

FROM golang:1.25.6-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/producer ./cmd/producer

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app

COPY --from=build /out/producer ./producer
COPY config.json ./

ENTRYPOINT ["./producer"]
