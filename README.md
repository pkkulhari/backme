# BackMe

A CLI tool for backing up databases and directories to Amazon S3.

## Features

- Backup PostgreSQL databases to S3
- Backup directories to S3 with optional sync and delete capabilities
- Scheduled backups via cron expressions
- Systemd service integration
- Simple YAML configuration

## Installation

### Automatic Installation

```bash
sudo curl -s https://raw.githubusercontent.com/pkkulhari/backme/master/install.sh | sudo bash
```

This will:

- Download the latest release
- Create a configuration file at `/etc/backme/config.yaml`
- Setup a systemd service
- Start the service

### Manual Installation

1. Download the latest release from [GitHub Releases](https://github.com/pkkulhari/backme/releases)
2. Make it executable: `chmod +x backme`
3. Copy it to your bin directory: `sudo cp backme /usr/local/bin/`
4. Create a config directory: `sudo mkdir -p /etc/backme`
5. Create a configuration file: `sudo cp config.example.yaml /etc/backme/config.yaml`

## Configuration

The configuration file is in YAML format:

```yaml
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

schedules:
  databases:
    - name: daily-backup
      expression: '0 0 * * *' # Run at midnight every day
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

  directories:
    - name: documents-backup
      expression: '0 0 * * *' # Run at midnight every day
      source_path: /path/to/your/documents
      sync: true
      delete: true
      aws:
        access_key_id: your-access-key
        secret_access_key: your-secret-key
        region: us-west-2
        bucket: your-bucket-name
        directory_prefix: documents
```

## Usage

### One-time Backup

#### Database Backup

```bash
backme db backup --db-name mydb --config /path/to/config.yaml
```

#### Directory Backup

```bash
backme dir backup --source /path/to/directory --sync --delete --config /path/to/config.yaml
```

Options:

- `--sync`: Only upload new or modified files
- `--delete`: Delete files from S3 that don't exist locally (only works with --sync)

### Scheduled Backups

Start the worker process to run scheduled backups:

```bash
backme worker --config /path/to/config.yaml
```

If installed as a service, you can manage it with systemd:

```bash
sudo systemctl start backme
sudo systemctl status backme
sudo systemctl stop backme
```

## Systemd Service

The installer creates a systemd service that runs the worker process. You can check its status with:

```bash
sudo systemctl status backme
```

View logs with:

```bash
sudo journalctl -u backme
```

## Building from Source

Requirements:

- Go 1.24 or later

Steps:

```bash
git clone https://github.com/pkkulhari/backme.git
cd backme
go build -o bin/backme cmd/*.go
```
