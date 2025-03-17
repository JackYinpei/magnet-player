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
  // return Promise.resolve({
  //   "filename": "某种物质",
  //   "year": 2024,
  //   "posterUrl": "https://image.tmdb.org/t/p/original/oDDYHINnemOisgswvLU0EZuHLFH.jpg",
  //   "backdropUrl": "https://image.tmdb.org/t/p/original/t98L9uphqBSNn2Mkvdm3xSFCQyi.jpg",
  //   "overview": "曾经红极一时的好莱坞巨星伊丽莎白无法面对自己老去的容颜，决定使用一种名为“完美物质”的黑市药物，透过注射药物的细胞复制物质，创造出更年轻、更好的另一个自己。“年华老去”及“年轻貌美”的自己该如何共存？会是更强烈的容貌焦虑大战？还是要不断迎合大众对“美”的期待？一场自我身体主导权的争夺战即将上演……",
  //   "rating": 7.1,
  //   "voteCount": 4204,
  //   "genres": [
  //     "恐怖",
  //     "科幻",
  //     "剧情",
  //     "喜剧"
  //   ],
  //   "runtime": 141,
  //   "tmdbId": 933260,
  //   "releaseDate": "2024-09-07",
  //   "originalTitle": "The Substance",
  //   "popularity": 6.078,
  //   "status": "Released",
  //   "tagline": "如果按照说明去做，会出什么问题？"
  // });
  // return Promise.resolve({
  //   "filename": "蜡笔小新：我们的恐龙日记",
  //   "year": 2024,
  //   "posterUrl": "https://image.tmdb.org/t/p/original/dTBhi2Y674JHaAktl550vpmxjF5.jpg",
  //   "backdropUrl": "https://image.tmdb.org/t/p/original/vW7lwVHkRePHzayZfoKOyYBeZqO.jpg",
  //   "overview": "在野原新之助的五岁暑假，东京新开了一个现代复活恐龙的主题公园，迎来了前所未有的恐龙热潮。而在春日部河滩旁，小白也偶遇到了一个新朋友——纳纳，随着春日部防卫队和纳纳的相处，恐龙公园背后的真相也随之揭露，巨大恐龙突然暴走街头，这次他们能否成功化解危机呢？",
  //   "rating": 5.7,
  //   "voteCount": 20,
  //   "genres": [
  //     "动画",
  //     "冒险",
  //     "喜剧",
  //     "家庭"
  //   ],
  //   "runtime": 106,
  //   "tmdbId": 1221404,
  //   "releaseDate": "2024-08-09",
  //   "originalTitle": "映画クレヨンしんちゃん オラたちの恐竜日記",
  //   "popularity": 0.826,
  //   "status": "Released"
  // });

  // 原始API调用方式（已注释）
  return fetchWithErrorHandling(`${API_BASE_URL}/search?filename=${encodeURIComponent(name)}`);
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