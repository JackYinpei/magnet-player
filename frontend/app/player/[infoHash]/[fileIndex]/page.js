'use client';

import React from 'react';
import { useState, useEffect, useCallback, useRef } from 'react';
import { useRouter } from 'next/navigation';
import { listFiles } from '@/lib/api';
import { VideoPlayer } from '@/components/video-player';
import { Button } from '@/components/ui/button';
import { Alert, AlertTitle, AlertDescription } from '@/components/ui/alert';
import { Card, CardContent } from '@/components/ui/card';

export default function PlayerPage({ params }) {
  const router = useRouter();
  // 使用 React.use() 解包 params
  const resolvedParams = React.use(params);
  const { infoHash, fileIndex } = resolvedParams;
  
  const [file, setFile] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [progress, setProgress] = useState(0);
  
  // 使用 ref 存储上一次的文件数据，用于比较避免不必要的渲染
  const prevFileRef = useRef(null);
  const updatingRef = useRef(false);
  const fileIdxRef = useRef(parseInt(fileIndex, 10));

  // 加载文件信息
  const loadFileInfo = useCallback(async (initialLoad = false) => {
    // 如果已经在更新中，跳过本次更新
    if (updatingRef.current && !initialLoad) return;
    
    try {
      updatingRef.current = true;
      
      if (initialLoad) {
        setLoading(true);
      }
      
      // 获取文件列表
      const files = await listFiles(infoHash);
      const fileIdx = fileIdxRef.current;
      
      // 找到指定的文件
      const currentFile = files.find(f => f.fileIndex === fileIdx);
      
      if (!currentFile) {
        setError('找不到指定的文件');
        return;
      }
      
      if (initialLoad) {
        // 检查文件是否是视频并且可播放
        if (!currentFile.isVideo) {
          setError('该文件不是视频文件');
          return;
        }
      }
      
      // 更新下载进度
      setProgress(Math.round(currentFile.progress * 100));
      
      // 避免在播放时不断重新设置文件，只在文件变化或初始加载时更新
      if (initialLoad || !prevFileRef.current || 
          JSON.stringify(currentFile) !== JSON.stringify(prevFileRef.current)) {
        prevFileRef.current = currentFile;
        setFile(currentFile);
      }
      
      // 只在非初始加载时，如果文件可播放状态发生变化才更新错误信息
      if (!initialLoad && prevFileRef.current) {
        if (!currentFile.isPlayable && prevFileRef.current.isPlayable) {
          setError('该视频文件暂时无法播放。请等待下载更多内容。');
        } else if (currentFile.isPlayable && !prevFileRef.current.isPlayable) {
          setError(null); // 文件可播放了，清除错误
        }
      } else if (initialLoad && !currentFile.isPlayable) {
        setError('该视频文件尚未准备好播放。请等待下载一部分内容后再尝试。');
      } else if (initialLoad) {
        setError(null);
      }
    } catch (err) {
      console.error('Failed to load file info:', err);
      setError('无法加载文件信息。请确保后端服务正在运行。');
    } finally {
      updatingRef.current = false;
      if (initialLoad) {
        setLoading(false);
      }
    }
  }, [infoHash]);
  
  // 初始加载
  useEffect(() => {
    if (infoHash && fileIndex !== undefined) {
      // 初始加载
      loadFileInfo(true);
      
      // 定期刷新进度（每5秒）
      const interval = setInterval(() => loadFileInfo(false), 5000);
      return () => clearInterval(interval);
    }
  }, [infoHash, fileIndex, loadFileInfo]);
  
  // 获取文件名（从路径中提取）
  const getFileName = (path) => {
    if (!path) return 'Unknown';
    return path.split('/').pop();
  };
  
  if (loading) {
    return <div className="text-center py-8">正在加载视频信息...</div>;
  }
  
  if (error && !file) {
    return (
      <div className="space-y-4">
        <Alert variant="destructive">
          <AlertTitle>错误</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
        <Button onClick={() => router.push(`/torrent/${infoHash}`)}>
          返回文件列表
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <h2 className="text-2xl font-bold">视频播放</h2>
        <div className="space-x-2">
          <Button 
            variant="outline" 
            onClick={() => router.push(`/torrent/${infoHash}`)}
          >
            返回文件列表
          </Button>
          <Button 
            variant="outline" 
            onClick={() => router.push('/')}
          >
            返回首页
          </Button>
        </div>
      </div>
      
      {file && (
        <>
          {error && (
            <Alert variant="warning" className="mb-4">
              <AlertTitle>警告</AlertTitle>
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}
          
          <VideoPlayer 
            infoHash={infoHash} 
            fileIndex={parseInt(fileIndex, 10)} 
            fileName={getFileName(file.path)} 
          />
          
          <Card>
            <CardContent className="pt-6">
              <h3 className="font-medium mb-2">文件信息</h3>
              <p className="text-sm text-muted-foreground mb-1">
                <strong>文件名:</strong> {getFileName(file.path)}
              </p>
              <p className="text-sm text-muted-foreground mb-1">
                <strong>路径:</strong> {file.path}
              </p>
              <p className="text-sm text-muted-foreground">
                <strong>下载进度:</strong> {progress}%
              </p>
            </CardContent>
          </Card>
        </>
      )}
    </div>
  );
}
