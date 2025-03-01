# Torrent Player

A web application that allows users to input a magnet link, list files within the torrent, and stream video files while they download.

## Features

- Add torrents via magnet links
- List all files in a torrent
- Stream video files while they're downloading (play-as-you-download)
- Modern UI using Next.js and Radix UI components
- Responsive design for desktop and mobile

## Project Structure

The project is divided into two main parts:

1. **Backend** (Golang): Handles torrent downloading and streaming
2. **Frontend** (Next.js): Provides the user interface

### Backend

The backend is built using Go and uses the [anacrolix/torrent](https://github.com/anacrolix/torrent) package for torrent operations.

Key components:
- `main.go`: Server entry point
- `torrent/client.go`: Wraps the torrent client functionality
- `api/handler.go`: HTTP handlers for the API endpoints

API Endpoints:
- `POST /api/magnet`: Add a magnet link
- `GET /api/torrents`: List all torrents
- `GET /api/files?infoHash={infoHash}`: List files in a torrent
- `GET /stream/{infoHash}/{fileIndex}`: Stream a file from a torrent

### Frontend

The frontend is built using Next.js with Radix UI components.

Key pages:
- Home page: Add torrents and view file lists
- Player page: Stream video files

## Getting Started

### Prerequisites

- Go 1.19 or later
- Node.js 16 or later
- npm

### Installation

1. Clone the repository
   ```bash
   git clone https://github.com/yourusername/torrentplayer.git
   cd torrentplayer
   ```

2. Set up the backend
   ```bash
   cd backend
   go mod download
   go run main.go
   ```

3. Set up the frontend
   Follow the instructions in `frontend-setup.md`

## Usage

1. Open the web application in your browser (default: http://localhost:3000)
2. Paste a magnet link in the input field and click "Add Torrent"
3. Once the torrent metadata is fetched, the files will be listed
4. Click on a video file to start streaming

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Disclaimer

This application is designed for streaming legal content only. The developers are not responsible for any misuse of this software. Please respect copyright laws in your region.
