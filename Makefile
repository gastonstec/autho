SERVICE_NAME = main

deps:
	go get ./...
	go mod tidy -go=1.16 && go mod tidy -go=1.17

build: deps
	go build -o ${SERVICE_NAME}

run: build
	./${SERVICE_NAME}
