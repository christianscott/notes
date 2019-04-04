all:
	CGO_ENABLED=1 go build -o notes -v

migrate:
	./migrate.sh

run:
	gin --port=3000 --appPort=8080 run main.go
