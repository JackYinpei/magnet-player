'use client';

import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { 
  TorrentInfo, 
  FileInfo, 
  MovieDetails, 
  SearchFilters 
} from '@/types';

// 应用状态类型
interface AppState {
  // 种子相关状态
  torrents: TorrentInfo[];
  selectedTorrent: TorrentInfo | null;
  isLoadingTorrents: boolean;
  torrentError: string | null;
  
  // UI状态
  activeTab: string;
  sidebarOpen: boolean;
  
  // 播放器状态
  currentPlayingFile: {
    infoHash: string;
    fileIndex: number;
    fileName: string;
  } | null;
  
  // 搜索状态
  searchQuery: string;
  searchResults: any[];
  isSearching: boolean;
  searchFilters: SearchFilters;
  
  // Actions
  setTorrents: (torrents: TorrentInfo[]) => void;
  addTorrent: (torrent: TorrentInfo) => void;
  updateTorrent: (infoHash: string, updates: Partial<TorrentInfo>) => void;
  removeTorrent: (infoHash: string) => void;
  setSelectedTorrent: (torrent: TorrentInfo | null) => void;
  setLoadingTorrents: (loading: boolean) => void;
  setTorrentError: (error: string | null) => void;
  
  setActiveTab: (tab: string) => void;
  setSidebarOpen: (open: boolean) => void;
  
  setCurrentPlayingFile: (file: { infoHash: string; fileIndex: number; fileName: string } | null) => void;
  
  setSearchQuery: (query: string) => void;
  setSearchResults: (results: any[]) => void;
  setSearching: (searching: boolean) => void;
  setSearchFilters: (filters: Partial<SearchFilters>) => void;
  
  // 清除所有数据
  clearAll: () => void;
}

// 创建状态store
export const useAppStore = create<AppState>()(
  persist(
    (set, get) => ({
      // 初始状态
      torrents: [],
      selectedTorrent: null,
      isLoadingTorrents: false,
      torrentError: null,
      
      activeTab: 'torrents',
      sidebarOpen: false,
      
      currentPlayingFile: null,
      
      searchQuery: '',
      searchResults: [],
      isSearching: false,
      searchFilters: {},
      
      // Actions
      setTorrents: (torrents: TorrentInfo[]) => set({ torrents }),
      
      addTorrent: (torrent: TorrentInfo) => set((state) => ({
        torrents: [...state.torrents, torrent]
      })),
      
      updateTorrent: (infoHash: string, updates: Partial<TorrentInfo>) => set((state) => ({
        torrents: state.torrents.map(torrent =>
          torrent.infoHash === infoHash
            ? { ...torrent, ...updates }
            : torrent
        )
      })),
      
      removeTorrent: (infoHash: string) => set((state) => ({
        torrents: state.torrents.filter(torrent => torrent.infoHash !== infoHash),
        selectedTorrent: state.selectedTorrent?.infoHash === infoHash ? null : state.selectedTorrent
      })),
      
      setSelectedTorrent: (torrent: TorrentInfo | null) => set({ selectedTorrent: torrent }),
      setLoadingTorrents: (loading: boolean) => set({ isLoadingTorrents: loading }),
      setTorrentError: (error: string | null) => set({ torrentError: error }),
      
      setActiveTab: (tab: string) => set({ activeTab: tab }),
      setSidebarOpen: (open: boolean) => set({ sidebarOpen: open }),
      
      setCurrentPlayingFile: (file: { infoHash: string; fileIndex: number; fileName: string } | null) => 
        set({ currentPlayingFile: file }),
      
      setSearchQuery: (query: string) => set({ searchQuery: query }),
      setSearchResults: (results: any[]) => set({ searchResults: results }),
      setSearching: (searching: boolean) => set({ isSearching: searching }),
      setSearchFilters: (filters: Partial<SearchFilters>) => set((state) => ({
        searchFilters: { ...state.searchFilters, ...filters }
      })),
      
      clearAll: () => set({
        torrents: [],
        selectedTorrent: null,
        isLoadingTorrents: false,
        torrentError: null,
        currentPlayingFile: null,
        searchQuery: '',
        searchResults: [],
        isSearching: false,
        searchFilters: {}
      })
    }),
    {
      name: 'magnet-player-store',
      // 只持久化部分状态，排除加载状态和错误信息
      partialize: (state) => ({
        torrents: state.torrents,
        activeTab: state.activeTab,
        sidebarOpen: state.sidebarOpen,
        searchQuery: state.searchQuery,
        searchFilters: state.searchFilters
      })
    }
  )
);

// Hooks for specific state slices
export const useTorrents = () => useAppStore((state) => ({
  torrents: state.torrents,
  selectedTorrent: state.selectedTorrent,
  isLoadingTorrents: state.isLoadingTorrents,
  torrentError: state.torrentError,
  setTorrents: state.setTorrents,
  addTorrent: state.addTorrent,
  updateTorrent: state.updateTorrent,
  removeTorrent: state.removeTorrent,
  setSelectedTorrent: state.setSelectedTorrent,
  setLoadingTorrents: state.setLoadingTorrents,
  setTorrentError: state.setTorrentError
}));

export const useUI = () => useAppStore((state) => ({
  activeTab: state.activeTab,
  sidebarOpen: state.sidebarOpen,
  setActiveTab: state.setActiveTab,
  setSidebarOpen: state.setSidebarOpen
}));

export const usePlayer = () => useAppStore((state) => ({
  currentPlayingFile: state.currentPlayingFile,
  setCurrentPlayingFile: state.setCurrentPlayingFile
}));

export const useSearch = () => useAppStore((state) => ({
  searchQuery: state.searchQuery,
  searchResults: state.searchResults,
  isSearching: state.isSearching,
  searchFilters: state.searchFilters,
  setSearchQuery: state.setSearchQuery,
  setSearchResults: state.setSearchResults,
  setSearching: state.setSearching,
  setSearchFilters: state.setSearchFilters
}));