#!/bin/bash

# Exit on error
set -e

PROJECT_ROOT=$(pwd)
VAGRANT_DIR="$PROJECT_ROOT/deployments/vagrant"

echo "Deploying to Vagrant VMs..."

# Ensure we are in the vagrant directory for vagrant commands
cd "$VAGRANT_DIR"

echo "Ensuring Vagrant VMs are running..."
vagrant up

# Sync binaries to VMs using rsync
echo "Syncing binaries to lb01..."
vagrant ssh lb01 -c "mkdir -p /home/vagrant/cloud/bin"
rsync -avz -e "vagrant ssh lb01 --" "$PROJECT_ROOT/bin/lb" :"/home/vagrant/cloud/bin/lb"

echo "Syncing binaries to storage nodes..."
for node in storage01 storage02; do
    vagrant ssh $node -c "mkdir -p /home/vagrant/cloud/bin"
    rsync -avz -e "vagrant ssh $node --" "$PROJECT_ROOT/bin/storage" :"/home/vagrant/cloud/bin/storage"
done

# Stop any existing processes to avoid port conflicts
echo "Stopping any existing processes..."
vagrant ssh lb01 -c "sudo systemctl stop lb 2>/dev/null || true"
vagrant ssh lb01 -c "sudo pkill lb || true"
for node in storage01 storage02; do
    vagrant ssh $node -c "sudo systemctl stop storage 2>/dev/null || true"
    vagrant ssh $node -c "sudo pkill storage || true"
done

# Configure and start Load Balancer on lb01 as a systemd service
echo "Configuring lb service on lb01..."
vagrant ssh lb01 -c "sudo tee /etc/systemd/system/lb.service <<EOF
[Unit]
Description=Cloud Load Balancer
After=network.target

[Service]
ExecStart=/home/vagrant/cloud/bin/lb
WorkingDirectory=/home/vagrant/cloud
User=root
Group=root
Restart=always
StandardOutput=append:/home/vagrant/lb.log
StandardError=append:/home/vagrant/lb.log

[Install]
WantedBy=multi-user.target
EOF"
vagrant ssh lb01 -c "sudo systemctl daemon-reload && sudo systemctl enable --now lb"

# Configure and start Storage Service on storage nodes
nodes=("storage01" "storage02")
ips=("192.168.57.11" "192.168.57.12")

for i in "${!nodes[@]}"; do
    node=${nodes[$i]}
    ip=${ips[$i]}
    echo "Configuring storage service on $node ($ip)..."
    vagrant ssh $node -c "sudo tee /etc/systemd/system/storage.service <<EOF
[Unit]
Description=Cloud Storage Service
After=network.target

[Service]
Environment=NODE_IP=$ip
ExecStart=/home/vagrant/cloud/bin/storage
WorkingDirectory=/home/vagrant/cloud
User=vagrant
Group=vagrant
Restart=always
StandardOutput=append:/home/vagrant/storage.log
StandardError=append:/home/vagrant/storage.log

[Install]
WantedBy=multi-user.target
EOF"
    vagrant ssh $node -c "sudo systemctl daemon-reload && sudo systemctl enable --now storage"
done

echo "Deployment completed successfully!"
echo "LB: http://192.168.56.10/"
echo "Storage 01: http://192.168.57.11:8081/"
echo "Storage 02: http://192.168.57.12:8081/"
