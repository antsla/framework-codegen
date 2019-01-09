FROM golang:latest

ADD . /home

WORKDIR /home

RUN go build codegen.go
RUN ./codegen api.go api_handlers.go && gofmt -w api_handlers.go
RUN go test -v