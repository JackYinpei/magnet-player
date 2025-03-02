import { formatFileSize, formatProgress } from '@/lib/api';
import { Card, CardContent } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Progress } from '@/components/ui/progress';

export function TorrentCard({ torrent }) {
  if (torrent === null || torrent === undefined) {
    return null;
  }

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

  return (
    <Card>
      <CardContent className="pt-6">
        <div className="space-y-4">
          <div className="flex justify-between items-start">
            <div>
              <h3 className="text-xl font-medium">{torrent.name}</h3>
              <p className="text-sm text-muted-foreground">
                Info Hash: <code className="text-xs">{torrent.infoHash}</code>
              </p>
            </div>
            <Badge variant={getStateBadgeVariant(torrent.state)}>
              {torrent.state}
            </Badge>
          </div>
          
          <div className="space-y-2">
            <div className="flex justify-between text-sm text-muted-foreground">
              <span>Progress: {formatProgress(torrent.progress)}</span>
              <span>{formatFileSize(torrent.downloaded)} / {formatFileSize(torrent.length)}</span>
            </div>
            <Progress value={torrent.progress * 100} />
          </div>
          
          <div className="grid grid-cols-2 gap-4 text-sm">
            <div>
              <p className="text-muted-foreground">添加时间</p>
              <p>{new Date(torrent.addedAt).toLocaleString()}</p>
            </div>
            <div>
              <p className="text-muted-foreground">文件数量</p>
              <p>{torrent.files?.length || 0} 个文件</p>
            </div>
            <div>
              <p className="text-muted-foreground">下载速度</p>
              <p>{formatFileSize(torrent.downloadSpeed || 0)}/s</p>
            </div>
            <div>
              <p className="text-muted-foreground">上传速度</p>
              <p>{formatFileSize(torrent.uploadSpeed || 0)}/s</p>
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
