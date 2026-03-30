# API Documentation

## Base URL

```
https://api.vaultdrift.example.com
```

## Authentication

All API requests (except public share links) require authentication via Bearer token:

```http
Authorization: Bearer <token>
```

### Login

```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "user",
  "password": "password"
}
```

**Response:**
```json
{
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "expires_at": 1640995200,
    "user_id": "user_abc123",
    "username": "user"
  }
}
```

### Logout

```http
POST /api/v1/auth/logout
Authorization: Bearer <token>
```

## Files

### List Files

```http
GET /api/v1/files?parent_id=&limit=50&offset=0
Authorization: Bearer <token>
```

**Query Parameters:**
- `parent_id` - Parent folder ID (empty for root)
- `limit` - Number of items (default: 50, max: 100)
- `offset` - Pagination offset

**Response:**
```json
{
  "data": {
    "files": [
      {
        "id": "file_abc123",
        "name": "document.pdf",
        "type": "file",
        "size": 1024567,
        "mime_type": "application/pdf",
        "parent_id": "folder_def456",
        "user_id": "user_abc123",
        "created_at": 1640995200,
        "updated_at": 1640998800,
        "version": 1
      }
    ],
    "limit": 50,
    "offset": 0
  }
}
```

### Get File Details

```http
GET /api/v1/files/{id}
Authorization: Bearer <token>
```

### Create File Entry

```http
POST /api/v1/files
Authorization: Bearer <token>
Content-Type: application/json

{
  "parent_id": "folder_def456",
  "name": "document.pdf",
  "mime_type": "application/pdf",
  "size": 1024567
}
```

### Rename File

```http
PUT /api/v1/files/{id}
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "new_name.pdf"
}
```

### Move File

```http
PUT /api/v1/files/{id}
Authorization: Bearer <token>
Content-Type: application/json

{
  "parent_id": "new_folder_id"
}
```

### Delete File

```http
DELETE /api/v1/files/{id}
Authorization: Bearer <token>
```

Moves file to trash (soft delete).

## Folders

### Create Folder

```http
POST /api/v1/folders
Authorization: Bearer <token>
Content-Type: application/json

{
  "parent_id": "",
  "name": "My Documents"
}
```

### Get Folder Details

```http
GET /api/v1/folders/{id}
Authorization: Bearer <token>
```

### Get Breadcrumbs

```http
GET /api/v1/folders/{id}/breadcrumbs
Authorization: Bearer <token>
```

**Response:**
```json
{
  "data": {
    "breadcrumbs": [
      {"id": "", "name": "root"},
      {"id": "folder_123", "name": "Documents"},
      {"id": "folder_456", "name": "Projects"}
    ]
  }
}
```

## Upload

### Get Upload URL

```http
POST /api/v1/uploads
Authorization: Bearer <token>
Content-Type: application/json

{
  "parent_id": "folder_123",
  "name": "file.pdf",
  "mime_type": "application/pdf",
  "size": 1234567
}
```

**Response:**
```json
{
  "data": {
    "upload_url": "http://localhost:8080/internal/upload/abc123",
    "file_id": "file_def789"
  }
}
```

### Upload File

```http
PUT {upload_url}
Content-Type: application/pdf

<binary data>
```

## Download

### Get Download URL

```http
GET /api/v1/downloads/{file_id}
Authorization: Bearer <token>
```

**Response:**
```json
{
  "data": {
    "download_url": "http://localhost:8080/internal/download/abc123",
    "filename": "file.pdf",
    "expires_at": 1640998800
  }
}
```

### Download File

```http
GET {download_url}
```

## Sharing

### Create Share Link

```http
POST /api/v1/files/{id}/shares
Authorization: Bearer <token>
Content-Type: application/json

{
  "share_type": "link",
  "expires_days": 7,
  "password": "optional_password",
  "max_downloads": 10,
  "allow_upload": false,
  "preview_only": false,
  "permission": "read"
}
```

**Response:**
```json
{
  "data": {
    "share": {
      "id": "share_abc123",
      "file_id": "file_def456",
      "share_type": "link",
      "token": "a1b2c3d4e5f6",
      "expires_at": 1641600000,
      "is_active": true
    },
    "share_url": "http://localhost:8080/s/a1b2c3d4e5f6"
  }
}
```

### List Shares

```http
GET /api/v1/files/{id}/shares
Authorization: Bearer <token>
```

### Revoke Share

```http
DELETE /api/v1/shares/{share_id}
Authorization: Bearer <token>
```

### Access Public Share

```http
GET /s/{token}
```

If password protected:

```http
POST /s/{token}
Content-Type: application/json

{
  "password": "share_password"
}
```

## Trash

### List Trashed Items

```http
GET /api/v1/trash?limit=50&offset=0
Authorization: Bearer <token>
```

### Restore Item

```http
POST /api/v1/trash/{id}/restore
Authorization: Bearer <token>
```

### Permanently Delete

```http
DELETE /api/v1/trash/{id}
Authorization: Bearer <token>
```

### Empty Trash

```http
DELETE /api/v1/trash
Authorization: Bearer <token>
```

## Search

### Search Files

```http
GET /api/v1/files/search?q=query&limit=50
Authorization: Bearer <token>
```

## Real-time Events

### Server-Sent Events (SSE)

```http
GET /api/v1/events
Authorization: Bearer <token>
Accept: text/event-stream
```

**Event Types:**
- `file:created` - File uploaded
- `file:updated` - File modified
- `file:deleted` - File moved to trash
- `file:moved` - File moved to different folder
- `folder:created` - Folder created
- `folder:deleted` - Folder deleted
- `share:created` - Share link created
- `share:revoked` - Share link revoked

### WebSocket

```
ws://localhost:8080/ws
Authorization: Bearer <token>
```

**Message Types:**
- `auth` - Authenticate connection
- `subscribe` - Subscribe to folder updates
- `unsubscribe` - Unsubscribe from folder
- `sync_request` - Request sync
- `event` - Real-time events
- `ping/pong` - Keepalive

## Errors

All errors follow this format:

```json
{
  "error": {
    "code": "invalid_request",
    "message": "Detailed error message",
    "status": 400
  }
}
```

**HTTP Status Codes:**
- `200` - Success
- `201` - Created
- `204` - No Content
- `400` - Bad Request
- `401` - Unauthorized
- `403` - Forbidden
- `404` - Not Found
- `409` - Conflict
- `429` - Too Many Requests
- `500` - Internal Server Error

## Rate Limits

- Authentication: 5 requests per minute per IP
- API requests: 100 requests per minute per user
- Upload: 100 MB per minute per user
- Download: 500 MB per minute per user
