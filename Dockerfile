FROM golang:1.14 as modules

ADD go.mod go.sum /m/
RUN cd /m && go mod download

FROM golang:1.14 as builder

COPY --from=modules /go/pkg /go/pkg

RUN mkdir -p /simple-url-shortener
ADD . /simple-url-shortener
WORKDIR /simple-url-shortener

RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
    go build -o ./bin/simple-url-shortener  ./cmd/main.go

FROM scratch

COPY --from=builder /simple-url-shortener/bin/simple-url-shortener /shortener
COPY --from=builder /simple-url-shortener/internal/database/migrations /internal/database/migrations

CMD ["/shortener"]