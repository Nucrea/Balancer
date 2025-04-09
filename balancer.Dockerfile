FROM golang:1.24-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
COPY internal ./internal
COPY cmd/balancer ./cmd/balancer

RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=bind,source=go.sum,target=go.sum \
    --mount=type=bind,source=go.mod,target=go.mod \
    go mod download -x

ENV GOCACHE=/root/.cache/go-build
RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=cache,target="/root/.cache/go-build" \
    --mount=type=bind,source=go.sum,target=go.sum \
    --mount=type=bind,source=go.mod,target=go.mod \
    go build -ldflags "-s -w" -o ./app ./cmd/balancer

RUN chmod +x ./app

FROM alpine:3.21 AS runner
WORKDIR /app

COPY --from=builder app/app app

ENV LOGFILE=/var/log/balancer.log
ENV ADDRS=127.0.0.1:8081
ENV PORT=8080

EXPOSE 8080

ENTRYPOINT ["./app"]