'use client';

import React from 'react';
import { useRouter } from 'next/navigation';
import { VideoPlayer } from '@/components/video-player';
import { Button } from '@/components/ui/button';

export default function PlayerPage({ params }) {
  const router = useRouter();
  // 使用 React.use() 解包 params
  const resolvedParams = React.use(params);
  const { infoHash, fileIndex } = resolvedParams;
  
  // Decode the file path
  const fileName = decodeURIComponent(fileIndex);
  
  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <h2 className="text-2xl font-bold">视频播放</h2>
        <div className="space-x-2">
          <Button 
            variant="outline" 
            onClick={() => router.push('/')}
          >
            返回首页
          </Button>
        </div>
      </div>
      
      <VideoPlayer 
        infoHash={infoHash} 
        fileIndex={fileIndex} 
        fileName={fileName} 
      />
    </div>
  );
}
