FROM golang:1

ADD . /go/src/github.com/tobyjsullivan/shifty
RUN go install github.com/tobyjsullivan/shifty/monitor

CMD /go/bin/monitor
