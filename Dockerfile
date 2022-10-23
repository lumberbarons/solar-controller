FROM golang:1.18-bullseye as build-api

WORKDIR /go/src/app
ADD . /go/src/app

RUN go get -d -v ./...

RUN go build -o /go/bin/epever-controller

FROM gcr.io/distroless/base-debian11

COPY --from=build-api /go/bin/epever-controller /
ENV GIN_MODE=release

CMD ["/epever-controller"]