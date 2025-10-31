FROM node:22 AS build-frontend

WORKDIR /app

COPY site/package*.json site/

RUN cd site && npm install

COPY site/ site/

RUN cd site && npm run build

FROM golang:1.24 AS build-backend

WORKDIR /go/src/app

COPY . .

COPY --from=build-frontend /app/site/build internal/static/build

RUN go get -d -v ./...

RUN go build -o /go/bin/solar-controller ./cmd/controller

FROM debian:trixie-slim

COPY --from=build-backend /go/bin/solar-controller /

ENV GIN_MODE=release

CMD ["/solar-controller"]