# Torrent Player API Documentation

This document describes the API endpoints provided by the Torrent Player backend service. The backend is built using Go with the `anacrolix/torrent` package and provides functionality for adding torrents via magnet links, listing torrents, listing files, and streaming video content.

## Base URL

All API endpoints are relative to the base URL:

```
http://localhost:8080
```

## Authentication

The API currently does not require authentication.

## CORS

All endpoints have CORS enabled with the following headers:

```
Access-Control-Allow-Origin: *
Access-Control-Allow-Methods: GET, POST, OPTIONS
Access-Control-Allow-Headers: Content-Type
```

## Data Models

### TorrentInfo

Represents information about a torrent.

```typescript
interface TorrentInfo {
  infoHash: string;     // Unique identifier for the torrent
  name: string;         // Name of the torrent
  length: number;       // Total size in bytes
  files: FileInfo[];    // Array of files in the torrent
  downloaded: number;   // Number of bytes downloaded
  progress: number;     // Download progress (0.0 to 1.0)
  state: string;        // Current state ("downloading", "completed", "stalled")
  addedAt: string;      // ISO timestamp when the torrent was added
}
```

### FileInfo

Represents information about a file within a torrent.

```typescript
interface FileInfo {
  path: string;         // File path within the torrent
  length: number;       // File size in bytes
  progress: number;     // Download progress for this file (0.0 to 1.0)
  fileIndex: number;    // Index of the file within the torrent
  torrentId: string;    // The infoHash of the parent torrent
  isVideo: boolean;     // Whether the file is a video file
  isPlayable: boolean;  // Whether the file has enough data to start playing
}
```

## API Endpoints

### 1. Add Magnet Link

Adds a new torrent using a magnet URI.

- **URL**: `/api/magnet`
- **Method**: `POST`
- **Content-Type**: `application/json`

#### Request Body

```json
{
  "magnetUri": "magnet:?xt=urn:btih:..."
}
```

#### Success Response

- **Code**: 200 OK
- **Content**: A `TorrentInfo` object representing the added torrent

#### Error Responses

- **Code**: 400 Bad Request
  - **Content**: `Invalid request body` - If the request body is malformed
- **Code**: 500 Internal Server Error
  - **Content**: `Failed to add magnet link: [error message]` - If there was an error adding the magnet link

#### Example

```javascript
// Request
fetch('http://localhost:8080/api/magnet', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
  },
  body: JSON.stringify({
    magnetUri: 'magnet:?xt=urn:btih:...'
  }),
})
.then(response => response.json())
.then(data => console.log(data));

// Response (example)
{
  "infoHash": "2a6f4a8c3b5d7e9f1c2d4e6f8a0b2c4d6e8f0a2c",
  "name": "Example Torrent",
  "length": 1073741824,
  "files": [
    {
      "path": "example.mp4",
      "length": 1073741824,
      "progress": 0.05,
      "fileIndex": 0,
      "torrentId": "2a6f4a8c3b5d7e9f1c2d4e6f8a0b2c4d6e8f0a2c",
      "isVideo": true,
      "isPlayable": true
    }
  ],
  "downloaded": 53687091,
  "progress": 0.05,
  "state": "downloading",
  "addedAt": "2025-03-01T12:00:00Z"
}
```

### 2. List Torrents

Returns a list of all torrents added to the client.

- **URL**: `/api/torrents`
- **Method**: `GET`

#### Success Response

- **Code**: 200 OK
- **Content**: An array of `TorrentInfo` objects

#### Example

```javascript
// Request
fetch('http://localhost:8080/api/torrents')
.then(response => response.json())
.then(data => console.log(data));

// Response (example)
[
  {
    "infoHash": "2a6f4a8c3b5d7e9f1c2d4e6f8a0b2c4d6e8f0a2c",
    "name": "Example Torrent 1",
    "length": 1073741824,
    "files": [...],
    "downloaded": 53687091,
    "progress": 0.05,
    "state": "downloading",
    "addedAt": "2025-03-01T12:00:00Z"
  },
  {
    "infoHash": "3b7e5f1c9d8a6b4e2f0d5c7a9b3e1f5d7c9b3a5e",
    "name": "Example Torrent 2",
    "length": 2147483648,
    "files": [...],
    "downloaded": 2147483648,
    "progress": 1.0,
    "state": "completed",
    "addedAt": "2025-03-01T11:30:00Z"
  }
]
```

### 3. List Files in a Torrent

Returns a list of all files in a specific torrent.

- **URL**: `/api/files`
- **Method**: `GET`
- **Query Parameters**: `infoHash=[string]` (required)

#### Success Response

- **Code**: 200 OK
- **Content**: An array of `FileInfo` objects

#### Error Responses

- **Code**: 400 Bad Request
  - **Content**: `Missing infoHash parameter` - If the infoHash parameter is not provided
- **Code**: 500 Internal Server Error
  - **Content**: `Failed to list files: [error message]` - If there was an error listing the files

#### Example

```javascript
// Request
fetch('http://localhost:8080/api/files?infoHash=2a6f4a8c3b5d7e9f1c2d4e6f8a0b2c4d6e8f0a2c')
.then(response => response.json())
.then(data => console.log(data));

// Response (example)
[
  {
    "path": "example/video.mp4",
    "length": 1073741824,
    "progress": 0.05,
    "fileIndex": 0,
    "torrentId": "2a6f4a8c3b5d7e9f1c2d4e6f8a0b2c4d6e8f0a2c",
    "isVideo": true,
    "isPlayable": true
  },
  {
    "path": "example/subtitle.srt",
    "length": 10240,
    "progress": 1.0,
    "fileIndex": 1,
    "torrentId": "2a6f4a8c3b5d7e9f1c2d4e6f8a0b2c4d6e8f0a2c",
    "isVideo": false,
    "isPlayable": false
  }
]
```

### 4. Stream File

Streams a file from a torrent. This endpoint supports range requests for seeking in videos.

- **URL**: `/stream/{infoHash}/{fileIndex}`
- **Method**: `GET`
- **URL Parameters**:
  - `infoHash`: The info hash of the torrent
  - `fileIndex`: The index of the file within the torrent (as returned by the list files endpoint)
- **Optional Headers**:
  - `Range`: Standard HTTP range header (e.g., `bytes=0-1023`)

#### Success Response

- **Code**: 200 OK or 206 Partial Content (if range request)
- **Content**: The requested file's binary data
- **Headers**:
  - `Content-Type`: The MIME type of the file (e.g., `video/mp4`)
  - `Content-Length`: The length of the response in bytes
  - `Accept-Ranges`: `bytes`
  - `Content-Range`: (Only for range requests) The range being served (e.g., `bytes 0-1023/1073741824`)

#### Error Responses

- **Code**: 400 Bad Request
  - **Content**: `Invalid path format` - If the URL format is incorrect
  - **Content**: `Invalid file index` - If the file index is not a valid number
  - **Content**: `File index out of range` - If the file index is out of range for the torrent
- **Code**: 404 Not Found
  - **Content**: `Torrent not found` - If the torrent with the specified info hash is not found
- **Code**: 500 Internal Server Error
  - **Content**: `Failed to seek: [error message]` - If there was an error seeking to the requested position

#### Example

```javascript
// Example of using the stream endpoint with a video element
const videoElement = document.createElement('video');
videoElement.controls = true;
videoElement.src = `http://localhost:8080/stream/2a6f4a8c3b5d7e9f1c2d4e6f8a0b2c4d6e8f0a2c/0`;
document.body.appendChild(videoElement);

// Example of downloading a file
function downloadFile(torrentId, fileIndex, fileName) {
  const a = document.createElement('a');
  a.href = `http://localhost:8080/stream/${torrentId}/${fileIndex}`;
  a.download = fileName;
  a.click();
}
```

## Utility Functions

### Format File Size

Formats a file size in bytes to a human-readable string.

```typescript
function formatFileSize(bytes: number): string {
  if (bytes === 0) return '0 Bytes';

  const k = 1024;
  const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));

  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}
```

### Format Progress

Formats a progress value (0.0 to 1.0) as a percentage string.

```typescript
function formatProgress(progress: number): string {
  return `${(progress * 100).toFixed(1)}%`;
}
```

## Video File Types

The backend recognizes the following file extensions as video files:

- `.mp4`
- `.mkv`
- `.avi`
- `.mov`
- `.wmv`
- `.flv`
- `.webm`
- `.m4v`
- `.mpg`, `.mpeg`
- `.3gp`

## Best Practices

1. **Polling**: Regularly poll the `/api/torrents` and `/api/files` endpoints to update the UI with the latest progress information. A reasonable interval is 3-5 seconds.

2. **Error Handling**: Always handle errors from the API gracefully and provide feedback to the user.

3. **Playback**: Files are marked as `isPlayable` when they have at least some data downloaded. However, playback may still buffer if the download speed is too slow or if the user seeks to a part that hasn't been downloaded yet.

4. **UI Feedback**: Show download progress for both torrents and individual files to give users feedback on download status.

5. **Mobile Support**: Ensure your UI is responsive and works well on mobile devices, as video playback is supported on most modern mobile browsers.
