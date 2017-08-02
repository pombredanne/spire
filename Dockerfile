FROM golang:1.8

RUN mkdir -p /go/src/github.com/superscale/spire/
WORKDIR /go/src/github.com/superscale/spire/
COPY . /go/src/github.com/superscale/spire/
RUN go get -v
RUN go build

RUN go get github.com/onsi/ginkgo/ginkgo
RUN go get github.com/onsi/gomega

CMD ["./spire"]
EXPOSE 1883
EXPOSE 1884
