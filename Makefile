.PHONY: build test clean build-frontend build-backend build-linux-arm64

build: build-frontend build-backend

build-frontend:
	cd site && npm install && npm run build
	rm -rf internal/static/build
	cp -r site/build internal/static/build

build-backend:
	go build -o bin/solar-controller ./cmd/controller

build-linux-arm64: build-frontend
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -trimpath -o bin/solar-controller-linux-arm64 ./cmd/controller

test:
	go test ./...

clean:
	rm -f bin/solar-controller
	rm -f bin/solar-controller-linux-arm64
	rm -rf site/build
	rm -rf internal/static/build
