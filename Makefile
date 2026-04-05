# Variables
BINARY_NAME=lb
BUILD_DIR=bin
SOURCE_DIR=cmd/lb
VAGRANT_DIR=deployments/vagrant

.PHONY: build clean up ssh-lb ssh-n1 ssh-n2 help

## build: Compile the Go binary for the Vagrant Linux environment
build:
	@echo "Building for Linux (amd64)..."
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME) $(SOURCE_DIR)/main.go

## up: Start the Vagrant infrastructure (3 nodes)
up:
	cd $(VAGRANT_DIR) && vagrant up

## ssh-lb: SSH into the Load Balancer node
ssh-lb:
	cd $(VAGRANT_DIR) && vagrant ssh lb01

## ssh-n1: SSH into Worker Node 01
ssh-n1:
	cd $(VAGRANT_DIR) && vagrant ssh node01

## ssh-n2: SSH into Worker Node 02
ssh-n2:
	cd $(VAGRANT_DIR) && vagrant ssh node02

## run-lb: Build and run the LB inside the VM (Assumes binary is synced)
run-lb: build
	@echo "Connect to lb01 and run: sudo /home/vagrant/cloud/bin/lb"
	cd $(VAGRANT_DIR) && vagrant ssh lb01 -c "sudo /home/vagrant/cloud/bin/lb"

## clean: Destroy VMs and remove binaries
clean:
	cd $(VAGRANT_DIR) && vagrant destroy -f
	rm -rf $(BUILD_DIR)/*

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'