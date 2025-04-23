#!/bin/bash

# Exit on error
set -e

for BINDIR in /usr/local/bin /usr/bin /bin; do
    echo $PATH | grep -q $BINDIR && break || continue
done

# Default values
INSTALL_DIR="$BINDIR/backme"
CONFIG_DIR="/etc/backme"
SERVICE_NAME="backme"
GITHUB_REPO="pkkulhari/backme"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    if [ $2 -eq 0 ]; then
        echo -e "${GREEN}✓ $1${NC}"
    else
        echo -e "${RED}✗ $1${NC}"
        exit 1
    fi
}

# Check if script is run as root
if [ "$EUID" -ne 0 ]; then 
    echo "Please run as root"
    exit 1
fi

# Clean up any existing installation
if [ -f "$INSTALL_DIR/backme" ]; then
    echo "Removing existing installation..."
    rm -f "$INSTALL_DIR/backme"
    systemctl stop "$SERVICE_NAME"
    systemctl disable "$SERVICE_NAME"
    rm -f "/etc/systemd/system/$SERVICE_NAME.service"
    userdel -r backme
    groupdel backme
fi

# Create necessary directories
mkdir -p "$INSTALL_DIR"
mkdir -p "$CONFIG_DIR"

# Download latest release
echo "Downloading latest release..."
LATEST_RELEASE=$(curl -s "https://api.github.com/repos/$GITHUB_REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
curl -L "https://github.com/$GITHUB_REPO/releases/download/$LATEST_RELEASE/backme" -o "$INSTALL_DIR/backme"
chmod +x "$INSTALL_DIR/backme"
print_status "Downloaded latest release" $?

# Create configuration file
echo "Creating configuration file..."
cat > "$CONFIG_DIR/config.yaml" << EOL
# Sample configuration file
database:
  host: localhost
  port: 5432
  user: postgres
  password: secret
  name: mydb

aws:
  access_key_id: your-access-key
  secret_access_key: your-secret-key
  region: us-west-2
  bucket: your-bucket-name
  database_prefix: database
  directory_prefix: directory

# schedules:
#   databases:
#     - name: daily-backup
#       expression: '0 0 * * *' # Run at midnight every day
#       database:
#         host: localhost
#         port: 5432
#         user: postgres
#         password: secret
#         name: mydb
#       aws:
#         access_key_id: your-access-key
#         secret_access_key: your-secret-key
#         region: us-west-2
#         bucket: your-bucket-name
#         database_prefix: database

#   directories:
#     - name: documents-backup
#       expression: '0 0 * * *' # Run at midnight every day
#       source_path: /path/to/your/documents
#       sync: true
#       delete: true
#       aws:
#         access_key_id: your-access-key
#         secret_access_key: your-secret-key
#         region: us-west-2
#         bucket: your-bucket-name
#         directory_prefix: documents
EOL
print_status "Created configuration file" $?

# Create systemd service
echo "Creating systemd service..."
cat > "/etc/systemd/system/$SERVICE_NAME.service" << EOL
[Unit]
Description=BackMe Worker Service
After=network.target

[Service]
Type=simple
User=backme
Group=backme
ExecStart=$INSTALL_DIR/backme --config $CONFIG_DIR/config.yaml
Restart=always
RestartSec=3

[Install]
WantedBy=default.target
EOL
print_status "Created systemd service" $?

# Create user and group
echo "Creating service user..."
id -u backme &>/dev/null || useradd -r -s /bin/false backme
print_status "Created service user" $?

# Set permissions
echo "Setting permissions..."
chown -R backme:backme "$INSTALL_DIR"
chown -R backme:backme "$CONFIG_DIR"
chmod 644 "$CONFIG_DIR/config.yaml"
chmod 644 "/etc/systemd/system/$SERVICE_NAME.service"
print_status "Set permissions" $?

# Reload systemd and start service
echo "Starting service..."
systemctl daemon-reload
systemctl enable "$SERVICE_NAME"
systemctl start "$SERVICE_NAME"
print_status "Service started" $?

echo -e "\n${GREEN}Installation completed successfully!${NC}"
echo "You can check the service status with: systemctl status $SERVICE_NAME"
echo "Configuration file is located at: $CONFIG_DIR/config.yaml"
echo "Logs can be viewed with: journalctl -u $SERVICE_NAME"
