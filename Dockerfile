FROM golang:1.6

RUN mkdir -p /go/src/app
WORKDIR /go/src/app

COPY . /go/src/app
RUN go-wrapper download
RUN go-wrapper install

EXPOSE 3000
VOLUME ["/go/src/app"]
CMD ["go-wrapper", "run"]
