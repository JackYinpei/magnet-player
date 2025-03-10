import { useState } from 'react';
import Link from 'next/link';
import Image from 'next/image';
import { formatFileSize, formatProgress, getMovieInfo } from '@/lib/api';
import { Card, CardHeader, CardTitle, CardContent, CardFooter } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Progress } from '@/components/ui/progress';
import { Button } from '@/components/ui/button';

export function TorrentList({ torrents = [], onTorrentSelected }) {
  const [selectedTorrentId, setSelectedTorrentId] = useState(null);
  const [moviesInfo, setMoviesInfo] = useState({});
  const [expandedOverviews, setExpandedOverviews] = useState({});

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

  const toggleOverview = (infoHash) => {
    setExpandedOverviews(prev => ({
      ...prev,
      [infoHash]: !prev[infoHash]
    }));
  };

  const fetchMovieInfo = async (torrent) => {
    try {
      // Original API call code (commented out to avoid charges)
      // const movie = await getMovieInfo(torrent.name);
      
      // Using test data instead
      const movie = {
        "filename": "蜡笔小新：我们的恐龙日记",
        "year": 2024,
        "posterUrl": "https://image.tmdb.org/t/p/original/dTBhi2Y674JHaAktl550vpmxjF5.jpg",
        "backdropUrl": "https://image.tmdb.org/t/p/original/vW7lwVHkRePHzayZfoKOyYBeZqO.jpg",
        "overview": "在野原新之助的五岁暑假，东京新开了一个现代复活恐龙的主题公园，迎来了前所未有的恐龙热潮。而在春日部河滩旁，小白也偶遇到了一个新朋友——纳纳，随着春日部防卫队和纳纳的相处，恐龙公园背后的真相也随之揭露，巨大恐龙突然暴走街头，这次他们能否成功化解危机呢？",
        "rating": 5.7,
        "voteCount": 20,
        "genres": [
          "动画",
          "冒险",
          "喜剧",
          "家庭"
        ],
        "runtime": 106,
        "tmdbId": 1221404,
        "releaseDate": "2024-08-09",
        "originalTitle": "映画クレヨンしんちゃん オラたちの恐竜日記",
        "popularity": 18.581,
        "status": "Released"
      };
      
      if (movie) {
        setMoviesInfo(prev => ({
          ...prev,
          [torrent.infoHash]: movie
        }));
      }
    } catch (error) {
      console.error("Failed to fetch movie info:", error);
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
      {torrents.map((torrent) => {
        const movie = moviesInfo[torrent.infoHash];
        const isExpanded = expandedOverviews[torrent.infoHash];
        
        return (
          <Card
            key={torrent.infoHash}
            className={`cursor-pointer hover:border-primary transition-colors ${selectedTorrentId === torrent.infoHash ? 'border-primary' : ''
              }`}
            onClick={() => handleTorrentClick(torrent)}
          >
            <CardHeader className="pb-2">
              <div className="flex justify-between items-start">
                <div className="flex-1">
                  <CardTitle className="text-xl truncate">
                    {movie ? `${movie.filename}${movie.year ? ` (${movie.year})` : ''}` : torrent.name}
                  </CardTitle>
                  {movie && movie.genres && (
                    <div className="flex flex-wrap gap-1 mt-1">
                      {movie.genres.map((genre, index) => (
                        <Badge key={index} variant="outline" className="text-xs">
                          {genre}
                        </Badge>
                      ))}
                      {movie.rating && (
                        <Badge variant="secondary" className="ml-2">
                          ⭐ {movie.rating.toFixed(1)}
                        </Badge>
                      )}
                    </div>
                  )}
                </div>
                <Badge variant={getStateBadgeVariant(torrent.state)} className="ml-2 shrink-0">
                  {torrent.state}
                </Badge>
              </div>
            </CardHeader>

            <CardContent>
              {movie && movie.posterUrl && (
                <div className="flex gap-4 mb-4">
                  <div className="relative w-24 h-36 shrink-0 overflow-hidden rounded">
                    <Image 
                      src={movie.posterUrl} 
                      alt={movie.filename || torrent.name}
                      fill
                      className="object-cover"
                      sizes="(max-width: 768px) 100px, 150px"
                    />
                  </div>
                  <div className="flex-1">
                    {movie.overview && (
                      <div className="mb-2">
                        <h4 className="text-sm font-medium mb-1">简介:</h4>
                        <p className="text-sm text-muted-foreground">
                          {isExpanded ? movie.overview : `${movie.overview.substring(0, 100)}...`}
                          <Button 
                            variant="link" 
                            className="p-0 h-auto text-xs ml-1" 
                            onClick={(e) => {
                              e.stopPropagation();
                              toggleOverview(torrent.infoHash);
                            }}
                          >
                            {isExpanded ? '收起' : '展开'}
                          </Button>
                        </p>
                      </div>
                    )}
                  </div>
                </div>
              )}

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

            <CardFooter className="pt-2 flex justify-between flex-wrap gap-2">
              {!movie ? (
                <Button 
                  variant="outline" 
                  size="sm" 
                  onClick={(e) => {
                    e.stopPropagation();
                    fetchMovieInfo(torrent);
                  }}
                >
                  获取电影详情
                </Button>
              ) : (
                <div></div> // Empty div for spacing when button is not shown
              )}
              <Link href={`/torrent/${torrent.infoHash}`} passHref legacyBehavior>
                <Button variant="outline" size="sm" as="a">
                  View Files
                </Button>
              </Link>
            </CardFooter>
          </Card>
        );
      })}
    </div>
  );
}
