'use client';

import { useState, useEffect, useRef } from 'react';
import { listTorrents } from '@/lib/api';
import { TorrentForm } from '@/components/torrent-form';
import { TorrentList } from '@/components/torrent-list';
import { Alert, AlertTitle, AlertDescription } from '@/components/ui/alert';
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs';

export default function Home() {
  const [torrents, setTorrents] = useState([]);  
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [activeTab, setActiveTab] = useState('torrents');
  const isInitialLoadRef = useRef(true);
  const prevTorrentsRef = useRef([]);
  
  // 加载种子列表
  const loadTorrents = async () => {
    try {
      // 仅在初始加载时设置 loading 为 true
      if (isInitialLoadRef.current) {
        setLoading(true);
      }
      
      const data = await listTorrents();
      const newTorrents = data || [];
      
      // 检查数据是否有变化，仅在数据变化时更新状态
      const hasChanged = JSON.stringify(newTorrents) !== JSON.stringify(prevTorrentsRef.current);
      if (hasChanged || isInitialLoadRef.current) {
        setTorrents(newTorrents);
        prevTorrentsRef.current = newTorrents;
      }
      
      // 只在有错误时或初始加载时重置错误状态
      if (error || isInitialLoadRef.current) {
        setError(null);
      }
    } catch (err) {
      console.error('Failed to load torrents:', err);
      setError('无法加载种子列表。请确保后端服务正在运行。');
      
      // 只在数据已存在或初始加载时重置种子列表
      if (prevTorrentsRef.current.length > 0 || isInitialLoadRef.current) {
        setTorrents([]);
        prevTorrentsRef.current = [];
      }
    } finally {
      // 仅在初始加载时更新 loading 状态
      if (isInitialLoadRef.current) {
        setLoading(false);
        isInitialLoadRef.current = false;
      }
    }
  };

  // 初始加载
  useEffect(() => {
    loadTorrents();
    
    // 定期刷新种子列表（每5秒）
    const interval = setInterval(loadTorrents, 5000);
    return () => clearInterval(interval);
  }, []);

  // 处理新添加的种子
  const handleTorrentAdded = (newTorrent) => {
    // 强制重新加载，并将初始加载状态设置为 true 以显示加载指示器
    isInitialLoadRef.current = true;
    loadTorrents();
  };

  return (
    <div className="max-w-4xl mx-auto">
      <h2 className="text-3xl font-bold tracking-tight mb-6">Torrent Player</h2>
      
      <Tabs defaultValue="torrents" className="mb-8" onValueChange={setActiveTab}>
        <TabsList className="mb-4">
          <TabsTrigger value="torrents">我的种子</TabsTrigger>
          <TabsTrigger value="add">添加种子</TabsTrigger>
        </TabsList>
        
        <TabsContent value="torrents" className="space-y-4">
          {loading ? (
            <div className="text-center py-8">正在加载种子列表...</div>
          ) : error ? (
            <Alert variant="destructive">
              <AlertTitle>错误</AlertTitle>
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          ) : (
            <TorrentList torrents={torrents} />
          )}
        </TabsContent>
        
        <TabsContent value="add">
          <div className="space-y-4">
            <h3 className="text-lg font-medium">添加磁力链接</h3>
            <p className="text-muted-foreground">
              粘贴一个磁力链接来添加一个新的种子。链接格式应该以 "magnet:?" 开头。
            </p>
            <TorrentForm onTorrentAdded={handleTorrentAdded} />
          </div>
        </TabsContent>
      </Tabs>
    </div>
  );
}
