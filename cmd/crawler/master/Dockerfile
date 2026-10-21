# Build from the repository root:
#   docker build -f cmd/crawler/master/Dockerfile -t prawler-master .

FROM golang:1.25.6-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/master ./cmd/crawler/master

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app

COPY --from=build /out/master ./master
COPY config.json blacklists.json ./
COPY web ./web

EXPOSE 5723
ENTRYPOINT ["./master"]
