FROM golang:1

ADD . /go/src/github.com/tobyjsullivan/shifty
RUN  go install github.com/tobyjsullivan/shifty/position

CMD /go/bin/position
