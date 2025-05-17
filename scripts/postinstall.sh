#!/bin/bash
# Install required applications
apt-get update
apt-get install -y $(cat /usr/share/agent/dependencies.txt)

# Create upload directory
mkdir -p /tmp/agent-uploads
chmod 777 /tmp/agent-uploads

# Reload systemd and enable service
systemctl daemon-reload
systemctl enable agent.service
systemctl start agent.service