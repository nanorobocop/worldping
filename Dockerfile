FROM golang:1.11
WORKDIR /usr/src/app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o bin/worldping

FROM alpine:latest
COPY --from=0 /usr/src/app/bin/worldping /go/bin/worldping
ENTRYPOINT [ "/go/bin/worldping" ]
