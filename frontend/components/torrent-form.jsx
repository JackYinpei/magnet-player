import { useState } from 'react';
import { addMagnet } from '@/lib/api';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { Alert, AlertTitle, AlertDescription } from '@/components/ui/alert';

export function TorrentForm({ onTorrentAdded }) {
  const [magnetUri, setMagnetUri] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [success, setSuccess] = useState(false);

  // 验证磁力链接格式是否有效
  const isValidMagnetUri = (uri) => {
    // 基本格式检查：以 magnet:? 开头，包含 xt=urn:btih: 部分
    return uri.trim().startsWith('magnet:?') && uri.includes('xt=urn:btih:');
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    
    // 清除之前的状态
    setError(null);
    setSuccess(false);
    
    // 验证磁力链接
    if (!magnetUri.trim()) {
      setError('请输入磁力链接');
      return;
    }
    
    if (!isValidMagnetUri(magnetUri)) {
      setError('磁力链接格式无效，请确保以 magnet:? 开头并包含有效的哈希');
      return;
    }
    
    try {
      setLoading(true);
      const newTorrent = await addMagnet(magnetUri);
      
      // 重置表单
      setMagnetUri('');
      setSuccess(true);
      
      // 通知父组件
      if (onTorrentAdded) {
        onTorrentAdded(newTorrent);
      }
    } catch (err) {
      console.error('Failed to add magnet link:', err);
      setError(err.message || '添加磁力链接失败。请确保链接有效并且后端服务正在运行。');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="w-full max-w-3xl mx-auto space-y-4">
      {error && (
        <Alert variant="destructive">
          <AlertTitle>错误</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}
      
      {success && (
        <Alert className="bg-green-50 border-green-500 text-green-700">
          <AlertTitle>成功</AlertTitle>
          <AlertDescription>磁力链接已成功添加</AlertDescription>
        </Alert>
      )}
      
      <form onSubmit={handleSubmit} className="space-y-4">
        <div className="flex flex-col sm:flex-row gap-2">
          <Input
            type="text"
            value={magnetUri}
            onChange={(e) => setMagnetUri(e.target.value)}
            placeholder="输入磁力链接 (magnet:?xt=urn:btih:...)"
            disabled={loading}
            className="flex-1"
          />
          <Button type="submit" disabled={loading}>
            {loading ? '添加中...' : '添加'}
          </Button>
        </div>
      </form>
    </div>
  );
}
