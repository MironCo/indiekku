#!/bin/bash

# Indiekku Deployment Script
# Usage: ./deploy.sh [droplet-ip]

# Configuration
DROPLET_IP=${1:-"#the droplet ip"}  # change this to the droplet IP
REMOTE_USER="root"
REMOTE_PATH="/root/indiekku"
SSH_KEY="#path to key"  # Change this to your key path

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${YELLOW}🚀 Deploying Indiekku server to $DROPLET_IP...${NC}"

# Check if we have the IP
if [ "$DROPLET_IP" = "YOUR_DROPLET_IP_HERE" ] || [ "$DROPLET_IP" = "#the droplet ip" ]; then
    echo -e "${RED}❌ Please set your droplet IP!${NC}"
    echo "Usage: ./deploy.sh your-droplet-ip"
    echo "Or edit the script to set a default IP"
    exit 1
fi

# Check if indiekku binary exists
if [ ! -f "indiekku" ]; then
    echo -e "${RED}❌ Binary 'indiekku' not found!${NC}"
    echo "Run 'make build' first to compile the binary"
    exit 1
fi

# Check if server directory has a Unity build
echo -e "${YELLOW}🔍 Checking for Unity server build...${NC}"
UNITY_BUILD=$(find server/ -maxdepth 1 \( -name "*.x86_64" -o -name "*.exe" \) 2>/dev/null | head -n 1)

if [ -z "$UNITY_BUILD" ]; then
    echo -e "${RED}❌ No Unity server build found in server/ directory!${NC}"
    echo "Please place your Unity server build (*.x86_64 or *.exe) in the server/ folder"
    exit 1
fi

echo -e "${GREEN}✅ Found Unity build: $(basename "$UNITY_BUILD")${NC}"

# Sync only what we need
echo -e "${YELLOW}📁 Syncing files to $DROPLET_IP...${NC}"

# Create remote directory
ssh -i "$SSH_KEY" $REMOTE_USER@$DROPLET_IP "mkdir -p $REMOTE_PATH"

# Sync indiekku binary
echo -e "${YELLOW}  → Syncing indiekku binary...${NC}"
rsync -avz --progress -e "ssh -i '$SSH_KEY'" \
    ./indiekku \
    $REMOTE_USER@$DROPLET_IP:$REMOTE_PATH/

# Sync Dockerfile
echo -e "${YELLOW}  → Syncing Dockerfile...${NC}"
rsync -avz --progress -e "ssh -i '$SSH_KEY'" \
    ./Dockerfile \
    $REMOTE_USER@$DROPLET_IP:$REMOTE_PATH/

# Sync server directory
echo -e "${YELLOW}  → Syncing server build...${NC}"
rsync -avz --progress -e "ssh -i '$SSH_KEY'" \
    ./server/ \
    $REMOTE_USER@$DROPLET_IP:$REMOTE_PATH/server/

# Check if rsync succeeded
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Files synced successfully!${NC}"

    # Stop any existing containers
    echo -e "${YELLOW}🛑 Stopping existing containers...${NC}"
    ssh -i "$SSH_KEY" $REMOTE_USER@$DROPLET_IP "docker stop \$(docker ps -q --filter name=unity-server) 2>/dev/null || true"

    # Run indiekku
    echo -e "${YELLOW}🚀 Starting Indiekku server...${NC}"
    ssh -i "$SSH_KEY" $REMOTE_USER@$DROPLET_IP "cd $REMOTE_PATH && chmod +x indiekku && ./indiekku"

    if [ $? -eq 0 ]; then
        echo -e "${GREEN}🎉 Deployment complete!${NC}"
        echo -e "${GREEN}Server should be running on $DROPLET_IP:7777${NC}"
        echo ""
        echo -e "${YELLOW}Useful commands:${NC}"
        echo "  View logs:    ssh -i '$SSH_KEY' $REMOTE_USER@$DROPLET_IP 'docker logs -f \$(docker ps -q --filter name=unity-server)'"
        echo "  Stop server:  ssh -i '$SSH_KEY' $REMOTE_USER@$DROPLET_IP 'docker stop \$(docker ps -q --filter name=unity-server)'"
        echo "  Server status: ssh -i '$SSH_KEY' $REMOTE_USER@$DROPLET_IP 'docker ps --filter name=unity-server'"
    else
        echo -e "${RED}❌ Failed to start server${NC}"
        exit 1
    fi
else
    echo -e "${RED}❌ File sync failed${NC}"
    exit 1
fi