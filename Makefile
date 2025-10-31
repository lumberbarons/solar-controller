.PHONY: build test clean build-frontend build-backend

build: build-frontend build-backend

build-frontend:
	cd site && npm install && npm run build
	rm -rf internal/static/build
	cp -r site/build internal/static/build

build-backend:
	go build -o bin/solar-controller ./cmd/controller

test:
	go test ./...

clean:
	rm -f bin/solar-controller
	rm -rf site/build
	rm -rf internal/static/build
