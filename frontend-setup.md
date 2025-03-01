# Frontend Setup Instructions

To create the Next.js frontend project, please run the following command in your terminal:

```bash
npx create-next-app@latest frontend --ts --eslint --tailwind --app --src-dir --import-alias "@/*"
```

After creating the project, navigate to the frontend directory and install the required dependencies:

```bash
cd frontend
npm install @radix-ui/react-dialog @radix-ui/react-progress @radix-ui/react-tabs @radix-ui/react-toast @radix-ui/react-tooltip
npm install clsx tailwind-merge lucide-react
```

## Project Structure

Once the frontend project is created, we'll implement the following files:

1. `src/app/page.tsx` - Main page with the torrent input form and file listing
2. `src/app/player/[infoHash]/[fileIndex]/page.tsx` - Video player page
3. `src/components/ui/TorrentForm.tsx` - Form for adding torrent magnet links
4. `src/components/ui/FileList.tsx` - Component for listing files in a torrent
5. `src/components/ui/VideoPlayer.tsx` - Custom video player component
6. `src/lib/api.ts` - API client for interacting with the backend

## Running the Project

After creating the frontend project and implementing the files, you can run both the backend and frontend:

### Backend
```bash
cd backend
go run main.go
```

### Frontend
```bash
cd frontend
npm run dev
```

Then access the application at http://localhost:3000
