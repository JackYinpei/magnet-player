import { useState } from 'react';
import Link from 'next/link';
import { formatFileSize, formatProgress } from '@/lib/api';
import { Card, CardHeader, CardTitle, CardContent, CardFooter } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Progress } from '@/components/ui/progress';
import { Button } from '@/components/ui/button';

export function TorrentList({ torrents = [], onTorrentSelected }) {  
  const [selectedTorrentId, setSelectedTorrentId] = useState(null);

  const getStateBadgeVariant = (state) => {
    switch (state) {
      case 'completed':
        return 'success';
      case 'downloading':
        return 'default';
      case 'stalled':
        return 'warning';
      default:
        return 'secondary';
    }
  };

  const handleTorrentClick = (torrent) => {
    setSelectedTorrentId(torrent.infoHash);
    if (onTorrentSelected) {
      onTorrentSelected(torrent);
    }
  };

  if (!torrents || torrents.length === 0) {
    return (
      <Card className="w-full mt-4">
        <CardContent className="pt-6">
          <p className="text-center text-muted-foreground">No torrents added yet</p>
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="space-y-4 mt-4">
      {torrents.map((torrent) => (
        <Card 
          key={torrent.infoHash}
          className={`cursor-pointer hover:border-primary transition-colors ${
            selectedTorrentId === torrent.infoHash ? 'border-primary' : ''
          }`}
          onClick={() => handleTorrentClick(torrent)}
        >
          <CardHeader className="pb-2">
            <div className="flex justify-between items-start">
              <CardTitle className="text-xl truncate">{torrent.name}</CardTitle>
              <Badge variant={getStateBadgeVariant(torrent.state)} className="ml-2">
                {torrent.state}
              </Badge>
            </div>
          </CardHeader>
          
          <CardContent>
            <div className="space-y-2">
              <div className="flex justify-between text-sm text-muted-foreground">
                <span>Progress: {formatProgress(torrent.progress)}</span>
                <span>{formatFileSize(torrent.downloaded)} / {formatFileSize(torrent.length)}</span>
              </div>
              <Progress value={torrent.progress * 100} />
              
              <div className="flex justify-between text-sm mt-2">
                <span className="text-muted-foreground">Added: {new Date(torrent.addedAt).toLocaleString()}</span>
                <span className="text-muted-foreground">{torrent.files?.length || 0} files</span>
              </div>
            </div>
          </CardContent>
          
          <CardFooter className="pt-2">
            <div className="flex justify-end w-full">
              <Link href={`/torrent/${torrent.infoHash}`} passHref legacyBehavior>
                <Button variant="outline" size="sm" as="a">
                  View Files
                </Button>
              </Link>
            </div>
          </CardFooter>
        </Card>
      ))}
    </div>
  );
}
