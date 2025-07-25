'use client';

import { useEffect } from 'react';
import { TorrentForm } from '@/components/torrent-form';
import { TorrentList } from '@/components/torrent-list';
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs';
import { useUI } from '@/lib/store';
import { useTorrentActions } from '@/lib/actions';

export default function Home() {
  const { activeTab, setActiveTab } = useUI();
  const { loadTorrents } = useTorrentActions();

  // 组件加载时获取种子列表
  useEffect(() => {
    loadTorrents();
  }, [loadTorrents]);

  // 处理新添加的种子
  const handleTorrentAdded = () => {
    // 添加成功后切换到种子列表页
    setActiveTab('torrents');
    // 重新加载种子列表以获取最新状态
    loadTorrents();
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
              粘贴一个磁力链接来添加一个新的种子。链接格式应该以 "magnet:?" 开头。
            </p>
            <TorrentForm onTorrentAdded={handleTorrentAdded} />
          </div>
        </TabsContent>
      </Tabs>
    </div>
  );
}
