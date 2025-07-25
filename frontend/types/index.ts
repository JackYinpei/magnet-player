// 全局类型定义

// 种子状态类型
export interface TorrentInfo {
  infoHash: string;
  name: string;
  length: number;
  files: FileInfo[];
  downloaded: number;
  progress: number;
  state: string;
  addedAt: string;
  movieDetails?: MovieDetails;
}

// 文件信息类型
export interface FileInfo {
  path: string;
  length: number;
  progress: number;
  fileIndex: number;
  torrentId: string;
  isVideo: boolean;
  isPlayable: boolean;
}

// 电影详情类型
export interface MovieDetails {
  filename?: string;
  year?: number;
  posterUrl?: string;
  backdropUrl?: string;
  overview?: string;
  rating?: number;
  voteCount?: number;
  genres?: string[];
  runtime?: number;
  tmdbId?: number;
  releaseDate?: string;
  originalTitle?: string;
  popularity?: number;
  status?: string;
  tagline?: string;
}

// API响应类型
export interface ApiResponse<T = any> {
  data?: T;
  error?: string;
  message?: string;
  code?: number;
}

// 错误类型
export interface AppError {
  message: string;
  code?: number;
  details?: any;
}

// UI状态类型
export interface UIState {
  activeTab: string;
  sidebarOpen: boolean;
  loading: boolean;
  error: string | null;
}

// 播放器状态类型
export interface PlayerState {
  currentPlayingFile: {
    infoHash: string;
    fileIndex: number;
    fileName: string;
  } | null;
  isPlaying: boolean;
  volume: number;
  currentTime: number;
  duration: number;
}

// 搜索状态类型
export interface SearchState {
  query: string;
  results: any[];
  isSearching: boolean;
  filters: SearchFilters;
}

// 搜索过滤器类型
export interface SearchFilters {
  genre?: string;
  year?: number;
  rating?: number;
  sortBy?: 'name' | 'date' | 'size' | 'progress';
  sortOrder?: 'asc' | 'desc';
}

// 组件Props类型
export interface TorrentFormProps {
  onTorrentAdded?: (torrent?: TorrentInfo) => void;
}

export interface TorrentListProps {
  torrents?: TorrentInfo[];
  onSelectTorrent?: (torrent: TorrentInfo) => void;
}

export interface VideoPlayerProps {
  infoHash: string;
  fileIndex: number;
  fileName: string;
  autoPlay?: boolean;
  controls?: boolean;
}

// 配置类型
export interface AppConfig {
  apiBaseUrl: string;
  enableLogging: boolean;
  debug: boolean;
  autoRefreshInterval: number;
}

// 事件类型
export type AppEvent = 
  | { type: 'TORRENT_ADDED'; payload: TorrentInfo }
  | { type: 'TORRENT_REMOVED'; payload: string }
  | { type: 'TORRENT_UPDATED'; payload: { infoHash: string; updates: Partial<TorrentInfo> } }
  | { type: 'PLAYER_STATE_CHANGED'; payload: Partial<PlayerState> }
  | { type: 'ERROR_OCCURRED'; payload: AppError };

// 钩子函数类型
export type UseAsyncState<T> = {
  data: T | null;
  loading: boolean;
  error: string | null;
  refetch: () => Promise<void>;
};

// 分页类型
export interface PaginationParams {
  page: number;
  limit: number;
  total?: number;
}

export interface PaginatedResponse<T> {
  data: T[];
  pagination: {
    page: number;
    limit: number;
    total: number;
    totalPages: number;
  };
}