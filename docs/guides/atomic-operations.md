---
page_title: "Atomic Operations"
subcategory: "Guides"
description: |-
  Understanding ACID guarantees in the FileManager provider - atomic writes, file locking, and checksum verification.
---

# Atomic Operations and ACID Guarantees

The FileManager provider implements ACID (Atomicity, Consistency, Isolation, Durability) principles to ensure data integrity during file operations. This guide explains how these guarantees work and how to use them effectively.

## What is ACID?

| Property | Description | FileManager Implementation |
|----------|-------------|---------------------------|
| **Atomicity** | Operations complete fully or not at all | Atomic writes via temp file + rename |
| **Consistency** | Data is always in a valid state | Checksum verification |
| **Isolation** | Concurrent operations don't interfere | File locking |
| **Durability** | Completed operations persist | fsync after write |

## Atomic Writes

### How Atomic Writes Work

By default, FileManager uses atomic writes:

1. **Write to temp file**: Content is written to a temporary file in the same directory
2. **Sync to disk**: `fsync()` ensures data is flushed to disk
3. **Atomic rename**: The temp file is renamed to the target path
4. **Cleanup**: If anything fails, the temp file is removed

This ensures that the target file is either:
- **Unchanged** (if the write fails)
- **Completely updated** (if the write succeeds)

You'll never see a partially written file.

### Using Atomic Writes

```hcl
resource "filemanager_file" "config" {
  path    = "/etc/app/config.json"
  content = jsonencode(local.config)

  atomic_write = true  # Default, but explicit for clarity
}
```

### Disabling Atomic Writes

In some cases (network filesystems, special file systems), you may need to disable atomic writes:

```hcl
resource "filemanager_file" "nfs_file" {
  path    = "/mnt/nfs/config.json"
  content = jsonencode(local.config)

  atomic_write = false  # Direct write
}
```

~> **Warning:** Disabling atomic writes can result in partially written files if the write operation fails.

## Checksum Verification

### How It Works

When `verify_checksum` is enabled:

1. **Calculate expected checksum** from the content to write
2. **Write the file** using atomic write
3. **Read back the file** and calculate its checksum
4. **Compare checksums** to verify integrity
5. **Fail if mismatch** and roll back the change

### Using Checksum Verification

```hcl
resource "filemanager_file" "critical_config" {
  path    = "/etc/app/config.json"
  content = jsonencode(local.config)

  verify_checksum = true  # Verify after writing
}

# Check the computed checksum
output "config_sha256" {
  value = filemanager_file.critical_config.sha256
}
```

### When to Use Checksum Verification

Use checksum verification for:
- Critical configuration files
- Files written to remote backends (SSH, S3)
- High-reliability environments
- Binary files where corruption would be catastrophic

### Checksum Data Source

You can also calculate checksums separately:

```hcl
data "filemanager_checksum" "verify" {
  path = "/etc/app/config.json"
}

output "file_checksums" {
  value = {
    md5    = data.filemanager_checksum.verify.md5
    sha256 = data.filemanager_checksum.verify.sha256
    sha512 = data.filemanager_checksum.verify.sha512
  }
}
```

## File Locking

### How Locking Works

FileManager implements file locking to prevent concurrent access:

1. **Acquire lock**: Before writing, acquire an exclusive lock
2. **Perform operation**: Write the file
3. **Release lock**: After completion, release the lock

### Provider-Level Locking

Enable locking at the provider level:

```hcl
provider "filemanager" {
  enable_locking = true
  lock_timeout   = "30s"  # Wait up to 30 seconds for lock
}
```

### Lock Timeout

If a lock cannot be acquired within the timeout period, the operation fails:

```hcl
provider "filemanager" {
  enable_locking = true
  lock_timeout   = "1m"  # Increase timeout for slow systems
}
```

### Lock Contention

If you see lock timeout errors, check for:
- Multiple Terraform runs targeting the same files
- Long-running operations holding locks
- Deadlocks from circular dependencies

## Backup and Recovery

### Automatic Backups

FileManager can create backups before modifying files:

```hcl
resource "filemanager_file" "important" {
  path    = "/etc/app/config.json"
  content = jsonencode(local.config)

  backup           = true
  backup_retention = 5  # Keep last 5 backups
}
```

Backups are stored as:
- `/etc/app/config.json.bak.1` (most recent)
- `/etc/app/config.json.bak.2`
- ... up to `backup_retention`

### Custom Backup Directory

```hcl
provider "filemanager" {
  backup_enabled   = true
  backup_retention = 10
  backup_dir       = "/var/backups/terraform"
}
```

### Recovery

To recover from a backup, you can:

1. **Manual recovery**: Copy the backup file to the original location
2. **Terraform import**: Import the backup as a new resource
3. **State manipulation**: Remove the resource from state and re-apply

## Combining ACID Features

### Maximum Safety Configuration

For critical files, combine all safety features:

```hcl
provider "filemanager" {
  atomic_writes    = true
  verify_checksum  = true
  enable_locking   = true
  lock_timeout     = "1m"
  backup_enabled   = true
  backup_retention = 5
}

resource "filemanager_file" "critical" {
  path    = "/etc/critical/config.json"
  content = jsonencode(local.critical_config)

  # Resource-level overrides if needed
  atomic_write    = true
  verify_checksum = true
  backup          = true
}
```

### Performance vs. Safety Trade-offs

| Feature | Safety | Performance Impact |
|---------|--------|-------------------|
| Atomic writes | High | Low (extra rename) |
| Checksum verification | High | Medium (read-after-write) |
| File locking | Medium | Variable (wait time) |
| Backups | High | Medium (copy operation) |

For development environments, you might disable some features:

```hcl
provider "filemanager" {
  atomic_writes   = true   # Always keep this
  verify_checksum = false  # Skip in dev
  enable_locking  = false  # Skip in dev
  backup_enabled  = false  # Skip in dev
}
```

## Error Handling

### Atomic Write Failures

If an atomic write fails:
1. The temporary file is removed
2. The original file remains unchanged
3. Terraform reports an error

```
Error: Failed to write file

  The atomic write operation failed. The original file
  /etc/app/config.json has not been modified.

  Reason: disk full
```

### Checksum Mismatch

If checksum verification fails:
1. The file is considered corrupt
2. Terraform reports a checksum mismatch error
3. You should investigate the cause

```
Error: Checksum verification failed

  The file /etc/app/config.json was written but the
  checksum does not match.

  Expected: abc123...
  Actual:   def456...

  This may indicate disk corruption or a race condition.
```

### Lock Timeout

If a lock cannot be acquired:

```
Error: Lock acquisition timeout

  Could not acquire lock for /etc/app/config.json
  within 30s. Another process may be holding the lock.
```

## Best Practices

### 1. Always Use Atomic Writes

Even if you don't think you need them:

```hcl
resource "filemanager_file" "any" {
  path         = "/path/to/file"
  content      = "content"
  atomic_write = true  # Always
}
```

### 2. Use Checksums for Critical Files

```hcl
resource "filemanager_file" "critical" {
  path            = "/etc/critical/config"
  content         = local.critical_content
  verify_checksum = true
}
```

### 3. Enable Backups for Mutable State

```hcl
resource "filemanager_file" "stateful" {
  path             = "/var/lib/app/state"
  content          = local.state
  backup           = true
  backup_retention = 10
}
```

### 4. Use Locking for Shared Files

```hcl
provider "filemanager" {
  enable_locking = true
  lock_timeout   = "2m"
}
```

### 5. Monitor File Operations

Output checksums for monitoring:

```hcl
output "config_checksums" {
  value = {
    path   = filemanager_file.config.path
    sha256 = filemanager_file.config.sha256
    size   = filemanager_file.config.size
  }
}
```

## Troubleshooting

### "Permission denied during atomic write"

The temporary file needs write permission in the target directory:

```hcl
resource "filemanager_directory" "app" {
  path       = "/etc/app"
  permission = "0755"
}

resource "filemanager_file" "config" {
  path    = "/etc/app/config.json"
  content = "{}"

  depends_on = [filemanager_directory.app]
}
```

### "Cross-device link" error

Atomic writes require the temp file and target to be on the same filesystem:

```hcl
# If /tmp is on a different filesystem:
resource "filemanager_file" "config" {
  path         = "/data/config.json"
  content      = "{}"
  atomic_write = false  # Disable for cross-device writes
}
```

### Lock files not cleaned up

If lock files persist after errors:

```bash
# Find and remove stale lock files
find /etc/app -name "*.lock" -mmin +60 -delete
```
