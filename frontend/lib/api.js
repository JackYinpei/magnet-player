// API client for the Torrent Player backend

const API_BASE_URL = process.env.NEXT_PUBLIC_BACKEND_API_URL ? process.env.NEXT_PUBLIC_BACKEND_API_URL : "http://localhost:8080/magnet";

// 通用请求处理器，增强错误处理
async function fetchWithErrorHandling(url, options = {}) {
  try {
    const response = await fetch(url, options);
    
    // 检查 HTTP 状态码
    if (!response.ok) {
      // 尝试获取错误消息
      let errorMessage;
      try {
        const errorData = await response.json();
        errorMessage = errorData.message || errorData.error || `Server responded with status: ${response.status}`;
      } catch {
        errorMessage = `Server responded with status: ${response.status}`;
      }
      
      throw new Error(errorMessage);
    }
    
    // 对于 204 No Content 返回 null
    if (response.status === 204) {
      return null;
    }
    
    return response.json();
  } catch (error) {
    // 网络错误或者解析错误
    if (error.name === 'TypeError' && error.message.includes('Failed to fetch')) {
      throw new Error('无法连接到服务器。请确保后端服务正在运行。');
    }
    
    // 重新抛出其他错误
    throw error;
  }
}

/**
 * 添加一个磁力链接
 * @param {string} magnetUri 磁力链接
 * @returns {Promise<Object>} 添加的种子信息
 */
export async function addMagnet(magnetUri) {
  if (!magnetUri || !magnetUri.trim()) {
    throw new Error('磁力链接不能为空');
  }
  
  if (!magnetUri.startsWith('magnet:?')) {
    throw new Error('无效的磁力链接格式');
  }
  
  return fetchWithErrorHandling(`${API_BASE_URL}/api/magnet`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ magnetUri }),
  });
}

/**
 * 获取所有种子的列表
 * @returns {Promise<Array>} 种子列表
 */
export async function listTorrents() {
  return fetchWithErrorHandling(`${API_BASE_URL}/api/torrents`);
}

/**
 * 获取指定种子的文件列表
 * @param {string} infoHash 种子的 info hash
 * @returns {Promise<Array>} 文件列表
 */
export async function listFiles(infoHash) {
  if (!infoHash) {
    throw new Error('Info hash 不能为空');
  }
  
  return fetchWithErrorHandling(`${API_BASE_URL}/api/files?infoHash=${infoHash}`);
}

/**
 * 获取视频流的 URL
 * @param {string} infoHash 种子的 info hash
 * @param {number} fileIndex 文件索引
 * @returns {string} 视频流的 URL
 */
export function getStreamUrl(infoHash, fileIndex) {
  return `${API_BASE_URL}/stream/${infoHash}/${fileIndex}`;
}

/**
 * 格式化文件大小
 * @param {number} bytes 字节数
 * @returns {string} 格式化的文件大小
 */
export function formatFileSize(bytes) {
  if (bytes === 0) return '0 B';
  if (!bytes || isNaN(bytes)) return 'Unknown';
  
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

/**
 * 格式化进度
 * @param {number} progress 进度值 (0-1)
 * @returns {string} 格式化的进度
 */
export function formatProgress(progress) {
  if (progress === undefined || progress === null || isNaN(progress)) {
    return '0%';
  }
  
  return `${Math.round(progress * 100)}%`;
}

/**
 * 获取电影信息
 * @param {string} name 种子名称
 * @returns {Promise<Object>} 电影信息
 */
export async function getMovieInfo(name) {
  return fetchWithErrorHandling(`${API_BASE_URL}/search?filename=${name}`);
}

/**
 * 保存电影详情信息到服务器
 * @param {string} infoHash 种子的 info hash
 * @param {Object} movie 电影详情信息
 * @returns {Promise<Object>} 保存结果
 */
export async function saveMovieDetails(infoHash, movie) {
  if (!infoHash) {
    throw new Error('Info hash 不能为空');
  }
  
  if (!movie) {
    throw new Error('电影详情不能为空');
  }
  
  return fetchWithErrorHandling(`${API_BASE_URL}/api/movie-details`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ infoHash, movie }),
  });
}