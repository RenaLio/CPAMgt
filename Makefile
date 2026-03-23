.PHONY: init
init:
	go install github.com/google/wire/cmd/wire@latest

.PHONY: wire-server
wire-server:
	wire gen ./cmd/server/wire

.PHONY: wire-migration
wire-migration:
	wire gen ./cmd/migration/wire

.PHONY: wire
wire: wire-server wire-migration

.PHONY: migration
migration:
	go run ./cmd/migration

.PHONY: server
server:
	go run ./cmd/server

.PHONY: build
build:
	go build -ldflags="-s -w" -trimpath -o ./bin/server ./cmd/server/

.PHONY: test
test:
	go test -v ./...

.PHONY: test-cover
test-cover:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

.PHONY: clean
clean:
	rm -rf ./bin/
	rm -f coverage.out coverage.html

.PHONY: docker-build
docker-build:
	docker build -f deploy/docker/Dockerfile \
		--build-arg APP_RELATIVE_PATH=./cmd/server \
		--build-arg GOPROXY=https://goproxy.cn,direct \
		-t cpa-mgt:latest .

.PHONY: docker-run
docker-run:
	docker run --rm -p 8080:8080 cpa-mgt:latest

.PHONY: docker
docker: docker-build docker-run

.PHONY: docker-compose-up
docker-compose-up:
	cd deploy/docker && docker-compose up -d

.PHONY: docker-compose-down
docker-compose-down:
	cd deploy/docker && docker-compose down

.PHONY: all
all: wire build