FROM golang:alpine

WORKDIR /go/src/github.com/freechessclub/ics-wsproxy
COPY . .

RUN go-wrapper download
RUN go-wrapper install

CMD ["go-wrapper", "run"]
