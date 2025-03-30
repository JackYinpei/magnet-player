'use client'
import { getMovieDetails, getMovieInfo, saveMovieDetails, listTorrents, saveTorrentData} from '@/lib/api';
import { useState, useEffect } from 'react';
import { Button } from './ui/button';
import Image from 'next/image';
import { formatFileSize, formatProgress } from '@/lib/utils';
import { Calendar, Clock, Film, Star, Users } from 'lucide-react';
import Link from 'next/link';
import { useInterval } from '@/lib/hooks';

const MovieCard = ({ movie: initialMovie }) => {
  const [movie, setMovie] = useState(initialMovie);
  
  const handleGetDetails = async () => {
    try {
      const details = await getMovieInfo(movie.name);
      console.log(details, "查看details", movie);
      
      setMovie({ ...movie, movieDetails: details });
      const resp = await saveMovieDetails(movie.infoHash, details)
      console.log(resp, "查看resp");
    } catch (error) {
      console.error("Failed to fetch movie details:", error);
    }
  };

  return (
    <div className="flex flex-col md:flex-row overflow-hidden bg-card border rounded-lg shadow-lg mb-6 hover:shadow-xl transition-shadow duration-300">
      {/* Movie Poster */}
      <div className="relative w-full md:w-64 h-80 flex-shrink-0">
        {movie?.movieDetails?.posterUrl ? (
          <img
            src={movie.movieDetails.posterUrl} 
            alt={movie.movieDetails.filename || movie.name}
            className="w-full h-full object-cover"
          />
        ) : (
          <div className="w-full h-full flex items-center justify-center bg-muted">
            <Film size={48} className="text-muted-foreground" />
            <span className="text-muted-foreground">No Poster</span>
          </div>
        )}
      </div>
      
      {/* Movie Info */}
      <div className="flex-1 p-4">
        <div className="flex flex-col h-full">
          {/* Title and Rating */}
          <div className="flex justify-between items-start mb-2">
            <h2 className="text-2xl font-bold text-foreground">
              {movie?.movieDetails?.filename || movie.name}
            </h2>
            {movie?.movieDetails?.rating && (
              <div className="flex items-center text-yellow-500">
                <Star className="fill-yellow-500 mr-1" size={18} />
                <span>{movie.movieDetails.rating}</span>
                {movie.movieDetails.voteCount && (
                  <span className="text-sm text-muted-foreground ml-1">({movie.movieDetails.voteCount})</span>
                )}
              </div>
            )}
          </div>
          
          {/* Release Year and Runtime */}
          {movie?.movieDetails && (
            <div className="flex flex-wrap gap-3 text-sm text-muted-foreground mb-3">
              {movie.movieDetails.releaseDate && (
                <div className="flex items-center">
                  <Calendar size={16} className="mr-1" />
                  <span>{movie.movieDetails.releaseDate}</span>
                </div>
              )}
              {movie.movieDetails.runtime && (
                <div className="flex items-center">
                  <Clock size={16} className="mr-1" />
                  <span>{movie.movieDetails.runtime} min</span>
                </div>
              )}
              {movie.movieDetails.status && (
                <div className="px-2 py-0.5 bg-primary/10 text-primary rounded-full">
                  {movie.movieDetails.status}
                </div>
              )}
            </div>
          )}
          
          {/* Genres */}
          {movie?.movieDetails?.genres && movie.movieDetails.genres.length > 0 && (
            <div className="flex flex-wrap gap-2 mb-3">
              {movie.movieDetails.genres.map((genre, index) => (
                <span key={index} className="px-2 py-1 bg-secondary text-secondary-foreground rounded-md text-xs">
                  {genre}
                </span>
              ))}
            </div>
          )}
          
          {/* Overview */}
          <p className="text-sm text-card-foreground mb-4 flex-grow">
            {movie?.movieDetails?.overview || movie.description || "No description available."}
          </p>
          
          {/* Files List */}
          {movie.files && movie.files.length > 0 && (
            <div className="mb-4">
              <h3 className="text-sm font-semibold mb-2">Files</h3>
              <div className="space-y-2">
                {movie.files.map((file, index) => (
                  <div 
                    key={index} 
                    className={`p-2 rounded-md text-sm border ${file.isVideo && file.isPlayable ? 'hover:bg-secondary/50 cursor-pointer' : 'opacity-70'}`}
                  >
                    {file.isVideo && file.isPlayable ? (
                      <Link href={`/player/${file.torrentId}/${encodeURIComponent(file.path)}`}>
                        <div className="flex flex-col">
                          <div className="flex justify-between mb-1">
                            <span className="font-medium truncate">{file.path.split('/').pop()}</span>
                            <span className="text-xs text-muted-foreground">{formatFileSize(file.length)}</span>
                          </div>
                          <div className="flex justify-between text-xs">
                            <span className={`${file.isVideo ? 'text-blue-500' : 'text-gray-500'}`}>
                              {file.isVideo ? 'Video' : 'Other'} • Progress: {formatProgress(file.progress)}
                            </span>
                            {file.isVideo && file.isPlayable && (
                              <span className="text-green-500">Playable</span>
                            )}
                          </div>
                        </div>
                      </Link>
                    ) : (
                      <div className="flex flex-col">
                        <div className="flex justify-between mb-1">
                          <span className="font-medium truncate">{file.path.split('/').pop()}</span>
                          <span className="text-xs text-muted-foreground">{formatFileSize(file.length)}</span>
                        </div>
                        <div className="flex justify-between text-xs">
                          <span className={`${file.isVideo ? 'text-blue-500' : 'text-gray-500'}`}>
                            {file.isVideo ? 'Video' : 'Other'} • Progress: {formatProgress(file.progress)}
                          </span>
                          {file.isVideo && !file.isPlayable && (
                            <span className="text-yellow-500">Not yet playable</span>
                          )}
                        </div>
                      </div>
                    )}
                  </div>
                ))}
              </div>
            </div>
          )}
          
          {/* Filename info (if available) */}
          {movie?.movieDetails?.filename && (
            <div className="text-xs text-muted-foreground mb-3 truncate">
              <span className="font-semibold">Filename:</span> {movie.movieDetails.filename}
            </div>
          )}
          
          {/* Get Details Button */}
          {!movie?.movieDetails && (
            <div className="mt-auto">
              <Button onClick={handleGetDetails} variant="outline" className="w-full">
                获取详情
              </Button>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export function TorrentList() {
  const [movies, setMovies] = useState([]);
  useEffect(() => {
    getMovieDetails().then((details) => {
      setMovies(details);
      console.log(details);
    });
  }, []);
  useInterval(() => {
    listTorrents().then((newTorrents) => {
      console.log(newTorrents);
      // Merge the new torrent data with existing movie details
      setMovies(prevMovies => {
        return newTorrents.map(newTorrent => {
          // Find matching movie in the previous state
          const existingMovie = prevMovies.find(m => m.infoHash === newTorrent.infoHash);
          // Preserve movieDetails from existing movie if available
          if (existingMovie && existingMovie.movieDetails) {
            return { ...newTorrent, movieDetails: existingMovie.movieDetails };
          }
          return newTorrent;
        });
      });
    });
  }, 5000);

  useInterval(()=>{
    console.log("开始保存种子数据到数据库");
    for (const movie of movies) {
      console.log("保存种子数据到数据库", movie);
      saveTorrentData(movie.infoHash, movie).then(()=>{
        console.log("种子数据已保存到数据库");
      })
    }
  }, 1000 * 60 * 1)
  return (
    <div className="container mx-auto py-6 px-4">
      <h1 className="text-3xl font-bold mb-6">影片列表</h1>
      <div className="grid grid-cols-1 gap-6">
        {movies.map((movie, i) => (
          <MovieCard key={i} movie={movie} />
        ))}
      </div>
    </div>
  )
}