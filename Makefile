all:
	CGO_ENABLED=1 go build -o notes -v

migrate:
	./migrate.sh
