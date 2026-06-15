FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
RUN go install github.com/pressly/goose/v3/cmd/goose@latest
COPY . .
RUN CGO_ENABLED=0 go build -o /out/server ./cmd/server

FROM alpine:3.20
RUN adduser -D -u 10001 app
COPY --from=build /out/server /server
COPY --from=build /go/bin/goose /usr/local/bin/goose
COPY migrations /migrations
COPY docker-entrypoint.sh /docker-entrypoint.sh
RUN chmod +x /docker-entrypoint.sh
USER app
EXPOSE 8080
ENTRYPOINT ["/docker-entrypoint.sh"]
