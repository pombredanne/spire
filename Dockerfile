FROM golang:1.8

RUN mkdir -p /go/src/github.com/superscale/spire/
WORKDIR /go/src/github.com/superscale/spire/
COPY . /go/src/github.com/superscale/spire/
RUN go get -v
RUN go build

CMD ["./spire"]
EXPOSE 1883
