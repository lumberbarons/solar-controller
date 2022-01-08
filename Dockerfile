FROM golang:1.17-bullseye as build

WORKDIR /go/src/app
ADD . /go/src/app

RUN go get -d -v ./...

RUN go build -o /go/bin/epever_controller

FROM gcr.io/distroless/base-debian11

COPY --from=build /go/bin/epever_controller /
ENV GIN_MODE=release

CMD ["/epever_controller"]