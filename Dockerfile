FROM golang:1.10.1-alpine3.7

RUN apk add --no-cache git

WORKDIR /go/src/app
COPY . .

RUN go get -d -v ./...
RUN go install -v ./...

EXPOSE 9090

CMD ["app"]
