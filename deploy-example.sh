#!/bin/bash

# Unity Server Deployment Script
# Usage: ./deploy.sh [droplet-ip]

# Configuration
DROPLET_IP=${1:-"#the droplet ip"}  # change this to the droplet IP
REMOTE_USER="root"
REMOTE_PATH="/root/unity-server"
LOCAL_PATH="./"
SSH_KEY="#path to key"  # Change this to your key path

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${YELLOW}🚀 Deploying Unity server to $DROPLET_IP...${NC}"

# Check if we have the IP
if [ "$DROPLET_IP" = "YOUR_DROPLET_IP_HERE" ]; then
    echo -e "${RED}❌ Please set your droplet IP!${NC}"
    echo "Usage: ./deploy.sh your-droplet-ip"
    echo "Or edit the script to set a default IP"
    exit 1
fi

# Sync files (excluding common unnecessary files)
echo -e "${YELLOW}📁 Syncing files...${NC}"
rsync -avz --progress \
    -e "ssh -i '$SSH_KEY'" \
    --exclude '.git' \
    --exclude '.DS_Store' \
    --exclude '*.log' \
    --exclude 'node_modules' \
    --exclude '.env' \
    $LOCAL_PATH $REMOTE_USER@$DROPLET_IP:$REMOTE_PATH/

# Check if rsync succeeded
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Files synced successfully!${NC}"
    
    # Build and restart on remote server
    echo -e "${YELLOW}🔨 Building and restarting server...${NC}"
    ssh -i "$SSH_KEY" $REMOTE_USER@$DROPLET_IP "cd $REMOTE_PATH && export PATH=\$PATH:/usr/local/go/bin && make run"
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}🎉 Deployment complete!${NC}"
        echo -e "${GREEN}Server should be running on $DROPLET_IP:7777${NC}"
    else
        echo -e "${RED}❌ Run failed${NC}"
        exit 1
    fi
else
    echo -e "${RED}❌ File sync failed${NC}"
    exit 1
fi