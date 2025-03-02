'use client';

import { useState, useEffect } from 'react';
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
  
  // 加载种子列表
  const loadTorrents = async () => {
    try {
      setLoading(true);
      const data = await listTorrents();
      setTorrents(data || []);  
      setError(null);
    } catch (err) {
      console.error('Failed to load torrents:', err);
      setError('无法加载种子列表。请确保后端服务正在运行。');
      setTorrents([]);  
    } finally {
      setLoading(false);
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
    loadTorrents(); // 重新加载完整列表
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
