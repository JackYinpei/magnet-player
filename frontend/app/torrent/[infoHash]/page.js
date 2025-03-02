'use client';

import React from 'react';
import { useState, useEffect, useCallback, useRef } from 'react';
import { useRouter } from 'next/navigation';
import { listFiles, listTorrents } from '@/lib/api';
import { FileList } from '@/components/file-list';
import { TorrentCard } from '@/components/torrent-card';
import { Button } from '@/components/ui/button';
import { Alert, AlertTitle, AlertDescription } from '@/components/ui/alert';

export default function TorrentDetailsPage({ params }) {
  const router = useRouter();
  // 使用 React.use() 解包 params
  const resolvedParams = React.use(params);
  const { infoHash } = resolvedParams;
  
  const [torrent, setTorrent] = useState(null);
  const [files, setFiles] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  
  // 使用 ref 存储上一次的数据，用于比较避免不必要的渲染
  const prevTorrentRef = useRef(null);
  const prevFilesRef = useRef([]);
  const updatingRef = useRef(false);

  // 加载种子信息和文件列表
  const loadData = useCallback(async (initialLoad = false) => {
    // 如果已经在更新中，跳过本次更新
    if (updatingRef.current && !initialLoad) return;
    
    try {
      updatingRef.current = true;
      
      if (initialLoad) {
        setLoading(true);
      }
      
      // 获取种子列表
      const torrents = await listTorrents();
      const currentTorrent = torrents.find(t => t.infoHash === infoHash);
      
      if (!currentTorrent) {
        setError('找不到该种子');
        return;
      }
      
      // 仅当种子信息发生变化时才更新状态，减少渲染
      if (JSON.stringify(currentTorrent) !== JSON.stringify(prevTorrentRef.current)) {
        prevTorrentRef.current = currentTorrent;
        setTorrent(currentTorrent);
      }
      
      // 获取文件列表
      const filesList = await listFiles(infoHash);
      
      // 仅当文件列表发生变化时才更新状态
      if (JSON.stringify(filesList) !== JSON.stringify(prevFilesRef.current)) {
        prevFilesRef.current = filesList;
        setFiles(filesList);
      }
      
      setError(null);
    } catch (err) {
      console.error('Failed to load torrent details:', err);
      setError('无法加载种子详情。请确保后端服务正在运行。');
    } finally {
      updatingRef.current = false;
      if (initialLoad) {
        setLoading(false);
      }
    }
  }, [infoHash]);
  
  // 初始加载
  useEffect(() => {
    if (infoHash) {
      // 初始加载
      loadData(true);
      
      // 定期刷新数据（每5秒）
      const interval = setInterval(() => loadData(false), 5000);
      return () => clearInterval(interval);
    }
  }, [infoHash, loadData]);
  
  if (loading) {
    return <div className="text-center py-8">正在加载种子详情...</div>;
  }
  
  if (error) {
    return (
      <div className="space-y-4">
        <Alert variant="destructive">
          <AlertTitle>错误</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
        <Button onClick={() => router.push('/')}>返回首页</Button>
      </div>
    );
  }

  return (
    <div className="space-y-8">
      <div className="flex justify-between items-center">
        <h2 className="text-2xl font-bold">种子详情</h2>
        <Button variant="outline" onClick={() => router.push('/')}>
          返回首页
        </Button>
      </div>
      
      {torrent && <TorrentCard torrent={torrent} />}
      
      <div className="space-y-4">
        <h3 className="text-xl font-bold">文件列表</h3>
        <FileList files={files} infoHash={infoHash} />
      </div>
    </div>
  );
}
