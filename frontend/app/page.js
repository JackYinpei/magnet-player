'use client';

import { useState, useRef } from 'react';
import { TorrentForm } from '@/components/torrent-form';
import { TorrentList } from '@/components/torrent-list';
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs';

export default function Home() {
  const [torrents, setTorrents] = useState([]);
  const [activeTab, setActiveTab] = useState('torrents');
  const isInitialLoadRef = useRef(true);

  // 处理新添加的种子
  const handleTorrentAdded = (newTorrent) => {
    // 如果有返回新种子，直接添加到列表中
    if (newTorrent) {
      setTorrents(prev => [...prev, newTorrent]);
    }
    
    // 添加成功后切换到种子列表页
    setActiveTab('torrents');
    
    // 强制重新加载，并将初始加载状态设置为 true 以显示加载指示器
    isInitialLoadRef.current = true;
  };

  return (
    <div className="max-w-4xl mx-auto">
      <h2 className="text-3xl font-bold tracking-tight mb-6">Torrent Player</h2>

      <Tabs defaultValue="torrents" className="mb-8" value={activeTab} onValueChange={setActiveTab}>
        <TabsList className="mb-4">
          <TabsTrigger value="torrents">我的种子</TabsTrigger>
          <TabsTrigger value="add">添加种子</TabsTrigger>
        </TabsList>

        <TabsContent value="torrents" className="space-y-4">
          <TorrentList />
        </TabsContent>

        <TabsContent value="add">
          <div className="space-y-4">
            <h3 className="text-lg font-medium">添加磁力链接</h3>
            <p className="text-muted-foreground">
              粘贴一个磁力链接来添加一个新的种子。链接格式应该以 &quot;magnet:?&quot; 开头。
            </p>
            <TorrentForm onTorrentAdded={handleTorrentAdded} />
          </div>
        </TabsContent>
      </Tabs>
    </div>
  );
}
