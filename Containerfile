FROM golang:alpine as build

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY *.go ./

RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o ./communautoFinderBot
RUN go clean -cache -testcache -fuzzcache -modcache

FROM alpine
RUN apk update && apk add --no-cache curl
COPY --from=build /app /app
WORKDIR /app
EXPOSE 8443 8444

HEALTHCHECK --interval=30s --timeout=30s --start-period=5s --retries=3 CMD curl http://localhost:8444/health || exit 1

CMD ./communautoFinderBot