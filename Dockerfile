FROM golang:1.9.4-alpine3.7 as builder
WORKDIR /go/src/github.com/gnur/booksing/
RUN apk add --no-cache git
RUN go get github.com/jteeuwen/go-bindata/...
RUN go get github.com/elazarl/go-bindata-assetfs/...
COPY static static
RUN go-bindata-assetfs static/...
COPY vendor vendor
COPY *.go ./
RUN go build -o app *.go

FROM alpine:latest  
CMD ["./app"]
COPY testdata/ /books/
COPY --from=builder /go/src/github.com/gnur/booksing/app /
