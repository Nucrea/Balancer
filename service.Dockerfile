FROM golang:1.23-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
COPY internal ./internal
COPY cmd/service ./cmd/service

RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=bind,source=go.sum,target=go.sum \
    --mount=type=bind,source=go.mod,target=go.mod \
    go mod download -x

ENV GOCACHE=/root/.cache/go-build
RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=cache,target="/root/.cache/go-build" \
    --mount=type=bind,source=go.sum,target=go.sum \
    --mount=type=bind,source=go.mod,target=go.mod \
    go build -ldflags "-s -w" -o ./app ./cmd/service

RUN chmod +x ./app

FROM alpine:3.21 AS runner
WORKDIR /app

COPY --from=builder app/app app

ENV PORT=8081
EXPOSE 8081

ENTRYPOINT ["./app"]