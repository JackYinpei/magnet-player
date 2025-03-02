import { useRef, useEffect } from 'react';
import { getStreamUrl } from '@/lib/api';

export function VideoPlayer({ infoHash, fileIndex, fileName }) {
  const videoRef = useRef(null);
  const streamUrl = getStreamUrl(infoHash, fileIndex);

  useEffect(() => {
    // If videoRef exists and the browser supports video element
    if (videoRef.current) {
      // Load the video
      videoRef.current.load();
    }
  }, [infoHash, fileIndex]);

  return (
    <div className="w-full aspect-video bg-black relative rounded-lg overflow-hidden">
      <video
        ref={videoRef}
        controls
        autoPlay
        className="w-full h-full"
        poster="/poster-placeholder.jpg"
      >
        <source src={streamUrl} />
        Your browser does not support the video tag.
      </video>
      
      <div className="absolute bottom-0 left-0 right-0 p-4 bg-gradient-to-t from-black/80 to-transparent pointer-events-none">
        <h2 className="text-white font-medium truncate">{fileName}</h2>
      </div>
    </div>
  );
}
