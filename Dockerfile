FROM golang:1.26 AS base

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

FROM base AS service-builder

ARG SERVICE=kitchen

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/app ./cmd/${SERVICE}

FROM base AS goose-builder

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/goose \
    github.com/pressly/goose/v3/cmd/goose

FROM alpine:3.21 AS service

RUN apk add --no-cache ca-certificates

COPY --from=service-builder /out/app /usr/local/bin/app

USER nobody

ENTRYPOINT ["/usr/local/bin/app"]

FROM alpine:3.21 AS migrator

COPY --from=goose-builder /out/goose /usr/local/bin/goose
COPY migrations /migrations

USER nobody

ENTRYPOINT ["/usr/local/bin/goose"]
CMD ["up"]
