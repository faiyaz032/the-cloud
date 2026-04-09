# Variables
BUILD_DIR=bin
LB_BINARY=lb
STORAGE_BINARY=storage
VAGRANT_DIR=deployments/vagrant

STORAGE01_ADDR=192.168.56.13:8081
STORAGE02_ADDR=192.168.56.14:8081

.PHONY: build build-lb build-storage up sync \
	ssh-lb ssh-compute01 ssh-compute02 ssh-storage01 ssh-storage02 \
	start-lb stop-lb start-storage01 start-storage02 start-storage stop-storage start-all \
	run-lb clean help

## build: Build LB + storage binaries for Linux VMs
build: build-lb build-storage

## build-lb: Build load balancer binary
build-lb:
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(LB_BINARY) ./cmd/lb

## build-storage: Build storage node binary
build-storage:
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(STORAGE_BINARY) ./cmd/storage

## up: Start all Vagrant VMs
up:
	cd $(VAGRANT_DIR) && vagrant up

## sync: Force sync project folder into VMs
sync:
	cd $(VAGRANT_DIR) && vagrant rsync

## ssh-lb: SSH into load balancer VM
ssh-lb:
	cd $(VAGRANT_DIR) && vagrant ssh lb01

## ssh-compute01: SSH into compute01 VM
ssh-compute01:
	cd $(VAGRANT_DIR) && vagrant ssh compute01

## ssh-compute02: SSH into compute02 VM
ssh-compute02:
	cd $(VAGRANT_DIR) && vagrant ssh compute02

## ssh-storage01: SSH into storage01 VM
ssh-storage01:
	cd $(VAGRANT_DIR) && vagrant ssh storage01

## ssh-storage02: SSH into storage02 VM
ssh-storage02:
	cd $(VAGRANT_DIR) && vagrant ssh storage02

## start-lb: Build, sync, and start LB on lb01 in background
start-lb: build-lb sync
	cd $(VAGRANT_DIR) && vagrant ssh lb01 -c "sudo pkill -x $(LB_BINARY) || true; nohup sudo /home/vagrant/cloud/bin/$(LB_BINARY) > /home/vagrant/lb.log 2>&1 < /dev/null &"

## stop-lb: Stop LB process on lb01
stop-lb:
	cd $(VAGRANT_DIR) && vagrant ssh lb01 -c "sudo pkill -x $(LB_BINARY) || true"

## start-storage01: Build, sync, and start storage server on storage01
start-storage01: build-storage sync
	cd $(VAGRANT_DIR) && vagrant ssh storage01 -c "pkill -x $(STORAGE_BINARY) || true; mkdir -p /home/vagrant/cloud-data && nohup env NODE_IP=$(STORAGE01_ADDR) /home/vagrant/cloud/bin/$(STORAGE_BINARY) > /home/vagrant/storage.log 2>&1 < /dev/null &"

## start-storage02: Build, sync, and start storage server on storage02
start-storage02: build-storage sync
	cd $(VAGRANT_DIR) && vagrant ssh storage02 -c "pkill -x $(STORAGE_BINARY) || true; mkdir -p /home/vagrant/cloud-data && nohup env NODE_IP=$(STORAGE02_ADDR) /home/vagrant/cloud/bin/$(STORAGE_BINARY) > /home/vagrant/storage.log 2>&1 < /dev/null &"

## start-storage: Build, sync, and start storage servers on both storage nodes
start-storage: start-storage01 start-storage02
	@echo "Storage servers started on $(STORAGE01_ADDR) and $(STORAGE02_ADDR)"

## stop-storage: Stop storage server process on both storage nodes
stop-storage:
	cd $(VAGRANT_DIR) && vagrant ssh storage01 -c "pkill -x $(STORAGE_BINARY) || true"
	cd $(VAGRANT_DIR) && vagrant ssh storage02 -c "pkill -x $(STORAGE_BINARY) || true"

## start-all: Start LB and both storage nodes
start-all: start-storage start-lb
	@echo "LB and storage services started"

## run-lb: Build, sync, and run LB in foreground on lb01
run-lb: build-lb sync
	cd $(VAGRANT_DIR) && vagrant ssh lb01 -c "sudo /home/vagrant/cloud/bin/$(LB_BINARY)"

## clean: Destroy VMs and remove binaries
clean:
	cd $(VAGRANT_DIR) && vagrant destroy -f
	rm -rf $(BUILD_DIR)/*

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@awk '/^## / { desc = substr($$0, 4); next } /^[a-zA-Z0-9_-]+:/ { split($$1, a, ":"); if (desc != "") { printf "\033[36m%-18s\033[0m %s\n", a[1], desc; desc = "" } }' $(MAKEFILE_LIST)