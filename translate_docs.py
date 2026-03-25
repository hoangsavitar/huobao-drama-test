content = """# Data Migration Service Documentation

## Overview

The Data Migration Service is designed to automatically download media and migrate database records where the `local_path` field is empty. This service executes automatically when the application starts, downloading files from remote URLs to local storage and updating the `local_path` field in the database.

## Features

- ✅ **Automatic Execution**: Runs automatically when the service starts without manual intervention.
- ✅ **Asynchronous Processing**: Executes in the background, ensuring it does not block the service startup process.
- ✅ **Multi-Table Support**: Covers multiple tables including Scenes, Characters, Video Generations, and Storyboards.
- ✅ **Smart Categorization**: Automatically categorizes and stores data into different directories based on their data type.
- ✅ **Fault Tolerance**: A download failure for a single file will not interrupt the processing of other files.
- ✅ **Detailed Logging**: Provides comprehensive execution logs and statistical information.

## Processed Data Tables

### 1. Scenes Table (`scenes`)
- **Field**: `image_url` → `local_path`
- **Storage Directory**: `data/storage/images/`
- **File Naming**: `scene_{id}_{timestamp}.{ext}`

### 2. Characters Table (`characters`)
- **Field**: `image_url` → `local_path`
- **Storage Directory**: `data/storage/characters/`
- **File Naming**: `character_{id}_{timestamp}.{ext}`

### 3. Video Generation Table (`video_generations`)
- **Field**: `video_url` → `local_path`
- **Storage Directory**: `data/storage/videos/`
- **File Naming**: `video_{id}_{timestamp}.{ext}`

### 4. Storyboards Table (`storyboards`)
- **Field**: `image_url` → `local_path`
- **Storage Directory**: `data/storage/images/`
- **File Naming**: `storyboard_{id}_{timestamp}.{ext}`

## Execution Flow

```text
1. Service Startup
   ↓
2. Database Connection & Migration
   ↓
3. Start Data Migration Task (Asynchronous)
   ↓
4. Create Storage Directories
   ↓
5. Query records across tables where local_path is empty
   ↓
6. Iterate through each record
   ├─ Download file locally
   ├─ Update local_path field
   └─ Record success/failure statistics
   ↓
7. Output Execution Statistics
```

## Configuration Guide

### Storage Root Directory
The default storage root is `data/storage`. This can be modified in the code:
```go
storageRoot: "data/storage"  // Customize path here
```

### Download Timeout Setting
The default HTTP request timeout is 60 seconds:
```go
client := &http.Client{
    Timeout: 60 * time.Second,  // Adjust based on network conditions
}
```

## Error Handling

### Skipped Scenarios
Files will be skipped under these conditions:
- URL is empty.
- URL is already a local path (starts with `/static/` or `data/`).
- HTTP request fails (e.g., 404 Not Found, Timeout).
- File writing fails.
- Database update fails.

### Errors Will NOT Cause:
- ❌ Service startup failure
- ❌ Interruption of other data processing tasks
- ❌ Database rollbacks

## Manual Triggering

If you need to trigger data migration manually (e.g., during runtime), you can do so as follows:

```go
// Create service instance
migrationService := services.NewDataMigrationService(db, logger)

// Execute migration
if err := migrationService.MigrateLocalPaths(); err != nil {
    log.Printf("Data migration failed: %v", err)
}
```

## Performance Considerations

- **Asynchronous Execution**: The batch task runs in the background and will not throttle server startup.
- **Network Bandwidth**: Downloading a large volume of files may consume significant bandwidth. It is recommended to limit concurrent downloads if handling a massive backlog.
- **Storage Space**: Ensure the server has adequate disk capacity to store the downloaded media.
- **Monitoring Suggestions**: Keep an eye on Failed Migrations > 10%, Execution Time > 5 minutes, and Disk Usage > 90%.

## Troubleshooting

### Issue: All Downloads Fail
- **Possible Causes**: Network connection issues, firewall blocking external requests, origin server is down.
- **Solutions**: Check server network connection, verify firewall settings, and test if the source URLs are accessible via `curl`.

### Issue: Partial Download Failures
- **Possible Causes**: Specific URLs have expired or are invalid (404/403), unsupported file formats, temporary network instability.
- **Solutions**: Check error logs for specific rejected URLs, manually verify URL validity.

### Issue: Database Updates Fail
- **Possible Causes**: Database connection lost, insufficient privileges, field constraints conflicts.
- **Solutions**: Verify DB connections and permissions.

## Code Locations
- **Service Implementation**: `application/services/data_migration_service.go`
- **Integration Setup**: `main.go` (Around lines 45-55)
- **Documentation File**: `docs/DATA_MIGRATION.md`
"""

with open("docs/DATA_MIGRATION.md", "w", encoding="utf-8") as f:
    f.write(content)
