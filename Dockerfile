FROM golang:1.12.7-alpine3.10
WORKDIR /usr/src/app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -o bin/worldping

FROM alpine:3.9
COPY --from=0 /usr/src/app/bin/worldping /go/bin/worldping
ENTRYPOINT [ "/go/bin/worldping" ]
