FROM golang:1.6

RUN mkdir -p /go/src/app
WORKDIR /go/src/app

EXPOSE 3000

COPY . /go/src/app
RUN go-wrapper download
RUN go-wrapper install

CMD ["go-wrapper", "run"]
