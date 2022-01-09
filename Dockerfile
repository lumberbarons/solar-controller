FROM node:16-alpine as build-site

WORKDIR /app
ENV PATH /app/node_modules/.bin:$PATH

COPY site/package.json ./
COPY site/package-lock.json ./

RUN npm ci --silent
RUN npm install react-scripts@3.4.1 -g --silent

COPY site/ ./
RUN npm run build

FROM golang:1.17-bullseye as build-api

WORKDIR /go/src/app
ADD . /go/src/app

RUN go get -d -v ./...

RUN go build -o /go/bin/epever_controller

FROM gcr.io/distroless/base-debian11

COPY --from=build-api /go/bin/epever_controller /
ENV GIN_MODE=release

COPY --from=build-site /app/build /site

CMD ["/epever_controller"]