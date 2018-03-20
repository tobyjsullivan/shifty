FROM golang:1

ADD . /go/src/github.com/tobyjsullivan/shifty
RUN  go install github.com/tobyjsullivan/shifty/tyche

CMD /go/bin/tyche
