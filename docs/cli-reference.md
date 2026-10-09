# Kumokura CLI Reference Manual

## General Syntax

```bash
kumokura [command] [subcommand] [flags]
```

### Global Flags
- `--account string`: Target storage account name (overrides default).
- `--json`: Output data in structured JSON format for scripting.

---

## Commands

### `account`
Manage storage accounts and access keys.

```bash
# List all accounts
kumokura account list [--json]

# Add a new account
kumokura account add --name prod-aws --type AWS --region us-east-1 \
  --access-key AKIA... --secret-key ...

# Add MinIO or self-hosted S3
kumokura account add --name local-minio --type MinIO --endpoint http://127.0.0.1:9000 \
  --path-style --access-key minioadmin --secret-key minioadmin

# Test account connectivity
kumokura account test <account-id>

# Remove an account
kumokura account remove <account-id>
```

### `bucket`
Manage S3 buckets.

```bash
# List buckets
kumokura bucket list

# Create a bucket
kumokura bucket create my-new-bucket --region us-east-1

# Delete a bucket
kumokura bucket delete my-new-bucket

# Check bucket region and location
kumokura bucket info my-new-bucket

# Configure versioning
kumokura bucket versioning my-new-bucket enable
kumokura bucket versioning my-new-bucket suspend
```

### `object`
Browse, upload, download, and delete S3 objects.

```bash
# List objects with prefix
kumokura object list my-bucket --prefix data/2026/

# Download object
kumokura object get my-bucket remote-file.zip ./local-file.zip

# Upload object
kumokura object put my-bucket remote-file.zip ./local-file.zip

# Copy or Move
kumokura object copy src-bkt file.txt dst-bkt file.txt
kumokura object move src-bkt old.txt src-bkt new.txt

# Delete object
kumokura object delete my-bucket file.txt

# Generate presigned URL (1 hour)
kumokura object presign my-bucket file.txt
```

### `sync`
High-performance directory synchronization.

```bash
# Dry run preview
kumokura sync ./local-folder my-bucket/prefix/ --dry-run

# Exact mirror sync (deletes extraneous destination files)
kumokura sync ./local-folder my-bucket/prefix/ --delete

# Filter patterns
kumokura sync ./local-folder my-bucket/prefix/ --exclude "*.tmp" --exclude ".git/**"
```
