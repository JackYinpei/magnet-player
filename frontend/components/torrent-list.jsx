import { useState, useEffect } from 'react';
import { Card, CardHeader, CardTitle, CardContent, CardFooter } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { PlayIcon, FolderIcon } from 'lucide-react';
import Image from 'next/image';
import { FileList } from './file-list';
import { Progress } from "@/components/ui/progress";
import { formatFileSize, formatProgress } from '@/lib/utils';
import { Skeleton } from '@/components/ui/skeleton';
import * as api from '@/lib/api';
import { useInterval } from '@/lib/hooks';

export function TorrentList() {
  const [torrents, setTorrents] = useState([]);
  const [selectedTorrent, setSelectedTorrent] = useState(null);
  const [files, setFiles] = useState([]);
  const [loading, setLoading] = useState(false);
  const [isExpanded, setIsExpanded] = useState(false);
  const [filesLoading, setFilesLoading] = useState(false);

  // Fetch torrents list on component mount
  useEffect(() => {
    fetchTorrents();
  }, []);

  // Poll for updates every 3 seconds
  useInterval(() => {
    fetchTorrents();
  }, 3000);

  const fetchTorrents = async () => {
    try {
      const data = await api.listTorrents();
      setTorrents(data);
    } catch (error) {
      console.error("Failed to fetch torrents:", error);
    }
  };

  const fetchMovieInfo = async (torrent) => {
    if (!torrent.name) return;
    
    try {
      setLoading(true);
      
      // Check if movie details already exist
      if (torrent.movieDetails) {
        setSelectedTorrent({ ...torrent });
        setLoading(false);
        return;
      }
      
      // Get movie info from API
      const movieInfo = await api.getMovieInfo(torrent.name);
      
      if (movieInfo && movieInfo.Title) {
        // Create movie details object
        const movieDetails = {
          title: movieInfo.Title,
          year: movieInfo.Year,
          poster: movieInfo.Poster,
          plot: movieInfo.Plot,
          genre: movieInfo.Genre,
          director: movieInfo.Director,
          actors: movieInfo.Actors,
          imdbRating: movieInfo.imdbRating,
          files: [] // Will be populated by the backend
        };
        
        // Update local state
        const updatedTorrent = { ...torrent, movieDetails };
        setSelectedTorrent(updatedTorrent);
        
        // Save movie details to server
        await api.saveMovieDetails(torrent.infoHash, movieDetails);
        
        // Update torrents list to show the movie details
        setTorrents(torrents.map(t => 
          t.infoHash === torrent.infoHash ? updatedTorrent : t
        ));
      } else {
        setSelectedTorrent(torrent);
      }
    } catch (error) {
      console.error('Error fetching movie info:', error);
      setSelectedTorrent(torrent);
    } finally {
      setLoading(false);
    }
  };

  const fetchFiles = async (torrent) => {
    try {
      setFilesLoading(true);
      const data = await api.listFiles(torrent.infoHash);
      setFiles(data);
    } catch (error) {
      console.error("Failed to fetch files:", error);
    } finally {
      setFilesLoading(false);
    }
  };

  const handleTorrentSelect = async (torrent) => {
    await fetchMovieInfo(torrent);
    await fetchFiles(torrent);
  };

  return (
    <div className="grid gap-4">
      <h2 className="text-2xl font-bold mb-4">My Torrents</h2>

      {torrents.length === 0 ? (
        <div className="text-center p-8">
          <p className="text-muted-foreground">No torrents added yet. Add a magnet link to get started.</p>
        </div>
      ) : (
        <div className="grid gap-4">
          {torrents.map((torrent) => (
            <Card key={torrent.infoHash} className="overflow-hidden">
              <CardHeader className="pb-2">
                <div className="flex justify-between items-start">
                  <div className="flex-1">
                    <CardTitle className="text-xl truncate">
                      {torrent.movieDetails ? 
                        `${torrent.movieDetails.title}${torrent.movieDetails.year ? ` (${torrent.movieDetails.year})` : ''}` 
                        : torrent.name}
                    </CardTitle>
                    {torrent.movieDetails && torrent.movieDetails.genre && (
                      <div className="flex flex-wrap gap-1 mt-1">
                        <Badge variant="outline" className="text-xs">
                          {torrent.movieDetails.genre}
                        </Badge>
                        {torrent.movieDetails.imdbRating && (
                          <Badge variant="secondary" className="ml-2">
                            ⭐ {torrent.movieDetails.imdbRating}
                          </Badge>
                        )}
                      </div>
                    )}
                  </div>
                </div>
              </CardHeader>

              <CardContent>
                {torrent.movieDetails && torrent.movieDetails.poster && (
                  <div className="flex gap-4 mb-4">
                    <div className="relative w-24 h-36 shrink-0 overflow-hidden rounded">
                      <Image 
                        src={torrent.movieDetails.poster} 
                        alt={torrent.movieDetails.title || torrent.name}
                        fill
                        className="object-cover"
                        sizes="(max-width: 768px) 100px, 150px"
                      />
                    </div>
                    <div className="flex-1">
                      {torrent.movieDetails.plot && (
                        <div className="mb-2">
                          <h4 className="text-sm font-medium mb-1">简介:</h4>
                          <p className="text-sm text-muted-foreground">
                            {isExpanded ? torrent.movieDetails.plot : `${torrent.movieDetails.plot.substring(0, 100)}...`}
                            {torrent.movieDetails.plot.length > 100 && (
                              <Button 
                                variant="link" 
                                className="p-0 h-auto text-xs ml-1" 
                                onClick={() => setIsExpanded(!isExpanded)}
                              >
                                {isExpanded ? "收起" : "展开"}
                              </Button>
                            )}
                          </p>
                        </div>
                      )}
                    </div>
                  </div>
                )}

                <div className="space-y-2">
                  <div className="flex justify-between text-sm">
                    <span>进度:</span>
                    <span>{formatProgress(torrent.progress)}</span>
                  </div>
                  <Progress value={torrent.progress * 100} className="h-2" />

                  <div className="grid grid-cols-2 gap-2 text-sm mt-2">
                    <div className="flex flex-col">
                      <span className="text-muted-foreground">大小:</span>
                      <span>{formatFileSize(torrent.length)}</span>
                    </div>
                    <div className="flex flex-col">
                      <span className="text-muted-foreground">已下载:</span>
                      <span>{formatFileSize(torrent.downloaded)}</span>
                    </div>
                  </div>
                </div>
              </CardContent>

              <CardFooter className="flex gap-2 pt-2">
                <Button 
                  variant="default" 
                  size="sm" 
                  onClick={() => handleTorrentSelect(torrent)}
                  className="w-full"
                >
                  <FolderIcon className="mr-2 h-4 w-4" />
                  浏览文件
                </Button>
              </CardFooter>
            </Card>
          ))}
        </div>
      )}

      {selectedTorrent && (
        <Card className="mt-6">
          <CardHeader>
            <CardTitle>
              {selectedTorrent.movieDetails?.title || selectedTorrent.name}
            </CardTitle>
          </CardHeader>
          <CardContent>
            {filesLoading ? (
              <div className="space-y-2">
                <Skeleton className="h-6 w-full" />
                <Skeleton className="h-6 w-full" />
                <Skeleton className="h-6 w-full" />
              </div>
            ) : (
              <FileList 
                files={files} 
                infoHash={selectedTorrent.infoHash} 
              />
            )}
          </CardContent>
        </Card>
      )}
    </div>
  );
}
