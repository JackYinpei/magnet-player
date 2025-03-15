'use client'
import { getMovieDetails, getMovieInfo, saveMovieDetails } from '@/lib/api';
import { useState, useEffect } from 'react';
import { TorrentCard } from './torrent-card';
import { Button } from './ui/button';
import Image from 'next/image';
import { formatFileSize, formatProgress } from '@/lib/utils';
import { Calendar, Clock, Film, Star, Users } from 'lucide-react';

const MovieCard = ({ movie: initialMovie }) => {
  const [movie, setMovie] = useState(initialMovie);
  
  const handleGetDetails = async () => {
    try {
      const details = await getMovieInfo(movie.name);
      setMovie({ ...movie, MovieDetails: details });
      await saveMovieDetails(movie.InfoHash, details)
    } catch (error) {
      console.error("Failed to fetch movie details:", error);
    }
  };

  return (
    <div className="flex flex-col md:flex-row overflow-hidden bg-card border rounded-lg shadow-lg mb-6 hover:shadow-xl transition-shadow duration-300">
      {/* Movie Poster */}
      <div className="relative w-full md:w-64 h-80 flex-shrink-0">
        {movie?.MovieDetails?.posterUrl ? (
          <img 
            src={movie.MovieDetails.posterUrl} 
            alt={movie.MovieDetails.originalTitle || movie.name}
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
              {movie?.MovieDetails?.originalTitle || movie.name}
            </h2>
            {movie?.MovieDetails?.rating && (
              <div className="flex items-center text-yellow-500">
                <Star className="fill-yellow-500 mr-1" size={18} />
                <span>{movie.MovieDetails.rating}</span>
                {movie.MovieDetails.voteCount && (
                  <span className="text-sm text-muted-foreground ml-1">({movie.MovieDetails.voteCount})</span>
                )}
              </div>
            )}
          </div>
          
          {/* Release Year and Runtime */}
          {movie?.MovieDetails && (
            <div className="flex flex-wrap gap-3 text-sm text-muted-foreground mb-3">
              {movie.MovieDetails.releaseDate && (
                <div className="flex items-center">
                  <Calendar size={16} className="mr-1" />
                  <span>{new Date(movie.MovieDetails.releaseDate).getFullYear()}</span>
                </div>
              )}
              {movie.MovieDetails.runtime && (
                <div className="flex items-center">
                  <Clock size={16} className="mr-1" />
                  <span>{movie.MovieDetails.runtime} min</span>
                </div>
              )}
              {movie.MovieDetails.status && (
                <div className="px-2 py-0.5 bg-primary/10 text-primary rounded-full">
                  {movie.MovieDetails.status}
                </div>
              )}
            </div>
          )}
          
          {/* Genres */}
          {movie?.MovieDetails?.genres && movie.MovieDetails.genres.length > 0 && (
            <div className="flex flex-wrap gap-2 mb-3">
              {movie.MovieDetails.genres.map((genre, index) => (
                <span key={index} className="px-2 py-1 bg-secondary text-secondary-foreground rounded-md text-xs">
                  {genre}
                </span>
              ))}
            </div>
          )}
          
          {/* Overview */}
          <p className="text-sm text-card-foreground mb-4 flex-grow">
            {movie?.MovieDetails?.overview || movie.description || "No description available."}
          </p>
          
          {/* Filename info (if available) */}
          {movie?.MovieDetails?.filename && (
            <div className="text-xs text-muted-foreground mb-3 truncate">
              <span className="font-semibold">Filename:</span> {movie.MovieDetails.filename}
            </div>
          )}
          
          {/* Get Details Button */}
          {!movie?.MovieDetails && (
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