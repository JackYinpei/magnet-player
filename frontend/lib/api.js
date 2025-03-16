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
 * 获取电影信息
 * @param {string} name 种子名称
 * @returns {Promise<Object>} 电影信息
 */
export async function getMovieInfo(name) {
  // 直接返回mock数据，确保字段名称与期望的完全一致
  return Promise.resolve({
    "backdropUrl": "https://via.placeholder.com/1280x720?text=%E3%80%90%E9%AB%98%E6%B8%85%E5%BD%B1%E8%A7%86%E4%B9%8B%E5%AE%B6%E5%8F%91%E5%B8%83+www.WHATMV.com%E3%80%91%E8%9C%A1%E7%AC%94%E5%B0%8F%E6%96%B0%EF%BC%9A%E6%88%91%E4%BB%AC%E7%9A%84%E6%81%90%E9%BE%99%E6%97%A5%E8%AE%B0%5B%E5%9B%BD%E6%97%A5%E5%A4%9A%E9%9F%B3%E8%BD%A8%2B%E4%B8%AD%E6%96%87%E5%AD%97%E5%B9%95%5D.2024.1080p.HamiVideo.WEB-DL.AAC2.0.H.264-DreamHD",
    "filename": "【高清影视之家发布 www.WHATMV.com】蜡笔小新：我们的恐龙日记[国日多音轨+中文字幕].2024.1080p.HamiVideo.WEB-DL.AAC2.0.H.264-DreamHD",
    "genres": [
      "未知"
    ],
    "originalTitle": "【高清影视之家发布 www.WHATMV.com】蜡笔小新：我们的恐龙日记[国日多音轨+中文字幕].2024.1080p.HamiVideo.WEB-DL.AAC2.0.H.264-DreamHD",
    "overview": "这是关于 【高清影视之家发布 www.WHATMV.com】蜡笔小新：我们的恐龙日记[国日多音轨+中文字幕].2024.1080p.HamiVideo.WEB-DL.AAC2.0.H.264-DreamHD 的电影简介。",
    "popularity": 1,
    "posterUrl": "https://via.placeholder.com/300x450?text=%E3%80%90%E9%AB%98%E6%B8%85%E5%BD%B1%E8%A7%86%E4%B9%8B%E5%AE%B6%E5%8F%91%E5%B8%83+www.WHATMV.com%E3%80%91%E8%9C%A1%E7%AC%94%E5%B0%8F%E6%96%B0%EF%BC%9A%E6%88%91%E4%BB%AC%E7%9A%84%E6%81%90%E9%BE%99%E6%97%A5%E8%AE%B0%5B%E5%9B%BD%E6%97%A5%E5%A4%9A%E9%9F%B3%E8%BD%A8%2B%E4%B8%AD%E6%96%87%E5%AD%97%E5%B9%95%5D.2024.1080p.HamiVideo.WEB-DL.AAC2.0.H.264-DreamHD",
    "rating": 5,
    "releaseDate": "2025-03-14",
    "runtime": 90,
    "status": "Released",
    "tmdbId": 0,
    "voteCount": 10,
    "year": 2024
  });

  // 原始API调用方式（已注释）
  // return fetchWithErrorHandling(`${API_BASE_URL}/api/search?filename=${encodeURIComponent(name)}`);
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

  return fetchWithErrorHandling(`${API_BASE_URL}/api/movie-details/${infoHash}`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(movie),
  });
}

/**
 * 获取所有电影详情信息
 * @returns {Promise<Array>} 电影详情列表
 */
export async function getMovieDetails() {
  return fetchWithErrorHandling(`${API_BASE_URL}/api/get-movie-details`);
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
 * 保存种子数据（包括文件路径）到数据库
 * @param {string} infoHash 种子的 info hash
 * @param {Object} torrentData 种子数据
 * @returns {Promise<Object>} 保存结果
 */
export async function saveTorrentData(infoHash, torrentData) {
  if (!infoHash) {
    throw new Error('Info hash 不能为空');
  }

  if (!torrentData) {
    throw new Error('种子数据不能为空');
  }

  return fetchWithErrorHandling(`${API_BASE_URL}/api/torrents/save-data/${infoHash}`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(torrentData),
  });
}