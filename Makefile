.PHONY: all build deploy clean

up: build deploy

build:
	@echo "Building binaries for Linux..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/lb cmd/lb/main.go
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/storage cmd/storage/*.go

deploy:
	@echo "Executing deployment script..."
	chmod +x scripts/deploy.sh
	./scripts/deploy.sh

clean:
	@echo "Cleaning up binaries..."
	rm -f bin/lb bin/storage
