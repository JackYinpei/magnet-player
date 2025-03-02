import { useState } from 'react';
import Link from 'next/link';
import { formatFileSize, formatProgress, getStreamUrl } from '@/lib/api';
import { Card, CardContent } from '@/components/ui/card';
import { Progress } from '@/components/ui/progress';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';

export function FileList({ files = [], infoHash }) {
  const [expandedFile, setExpandedFile] = useState(null);

  const toggleExpand = (fileIndex) => {
    setExpandedFile(expandedFile === fileIndex ? null : fileIndex);
  };

  if (!files || files.length === 0) {
    return (
      <Card className="w-full mt-4">
        <CardContent className="pt-6">
          <p className="text-center text-muted-foreground">No files found</p>
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="space-y-2 mt-4">
      {files.map((file) => (
        <Card 
          key={`${file.torrentId}-${file.fileIndex}`} 
          className="overflow-hidden"
        >
          <div 
            className="p-4 cursor-pointer hover:bg-muted/50 flex flex-col sm:flex-row justify-between"
            onClick={() => toggleExpand(file.fileIndex)}
          >
            <div className="flex-1 truncate">
              <div className="flex items-center">
                <span className="font-medium truncate">{file.path}</span>
                {file.isVideo && (
                  <Badge variant="secondary" className="ml-2">
                    Video
                  </Badge>
                )}
                {file.isPlayable && (
                  <Badge variant="success" className="ml-2">
                    Playable
                  </Badge>
                )}
              </div>
              <div className="flex text-sm mt-1 text-muted-foreground justify-between sm:justify-start">
                <span>{formatFileSize(file.length)}</span>
                <span className="sm:ml-4">Progress: {formatProgress(file.progress)}</span>
              </div>
            </div>
            
            <div className="flex mt-2 sm:mt-0 space-x-2">
              {file.isPlayable && (
                <Link href={`/player/${infoHash}/${file.fileIndex}`} passHref legacyBehavior>
                  <Button variant="default" size="sm" as="a">
                    Play
                  </Button>
                </Link>
              )}
              <a 
                href={getStreamUrl(infoHash, file.fileIndex)} 
                download={file.path.split('/').pop()}
                target="_blank"
                rel="noopener noreferrer"
              >
                <Button variant="outline" size="sm">
                  Download
                </Button>
              </a>
            </div>
          </div>
          
          <div className={`px-4 ${expandedFile === file.fileIndex ? 'pb-4' : 'h-0 overflow-hidden'}`}>
            <Progress value={file.progress * 100} />
          </div>
        </Card>
      ))}
    </div>
  );
}
