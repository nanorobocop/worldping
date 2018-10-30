FROM golang:1.11
WORKDIR /usr/src/app
RUN go get github.com/digineo/go-ping
RUN go get github.com/lib/pq
COPY worldping.go .
RUN CGO_ENABLED=0 GOOS=linux go build -o bin/worldping worldping.go

FROM alpine:latest
COPY --from=0 /usr/src/app/bin/worldping /go/bin/worldping
ENTRYPOINT [ "/go/bin/worldping" ]
