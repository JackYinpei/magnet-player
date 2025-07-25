'use client';

import { useCallback } from 'react';
import { useTorrents, useSearch, useAppStore } from './store';
import * as api from './api';

// 种子相关的异步操作hooks
export const useTorrentActions = () => {
  const {
    setTorrents,
    addTorrent,
    updateTorrent,
    removeTorrent,
    setLoadingTorrents,
    setTorrentError
  } = useTorrents();

  // 加载所有种子
  const loadTorrents = useCallback(async () => {
    try {
      setLoadingTorrents(true);
      setTorrentError(null);
      const torrents = await api.listTorrents();
      setTorrents(torrents || []);
    } catch (error) {
      console.error('加载种子列表失败:', error);
      setTorrentError(error.message || '加载种子列表失败');
    } finally {
      setLoadingTorrents(false);
    }
  }, [setTorrents, setLoadingTorrents, setTorrentError]);

  // 添加新种子
  const addNewTorrent = useCallback(async (magnetUri) => {
    try {
      setTorrentError(null);
      const newTorrent = await api.addMagnet(magnetUri);
      
      if (newTorrent) {
        addTorrent(newTorrent);
        
        // 保存种子数据到数据库
        try {
          await api.saveTorrentData(newTorrent.infoHash, newTorrent);
        } catch (saveErr) {
          console.error('保存种子数据失败:', saveErr);
          // 不中断流程，继续处理
        }
      }
      
      return newTorrent;
    } catch (error) {
      console.error('添加种子失败:', error);
      setTorrentError(error.message || '添加种子失败');
      throw error;
    }
  }, [addTorrent, setTorrentError]);

  // 删除种子
  const deleteTorrent = useCallback(async (infoHash) => {
    try {
      setTorrentError(null);
      // 这里可以添加删除API调用
      // await api.deleteTorrent(infoHash);
      removeTorrent(infoHash);
    } catch (error) {
      console.error('删除种子失败:', error);
      setTorrentError(error.message || '删除种子失败');
      throw error;
    }
  }, [removeTorrent, setTorrentError]);

  // 更新种子进度
  const updateTorrentProgress = useCallback(async (infoHash) => {
    try {
      // 这里可以添加获取最新进度的API调用
      // const updatedTorrent = await api.getTorrentProgress(infoHash);
      // updateTorrent(infoHash, updatedTorrent);
    } catch (error) {
      console.error('更新种子进度失败:', error);
    }
  }, [updateTorrent]);

  // 保存电影详情
  const saveMovieDetails = useCallback(async (infoHash, movieDetails) => {
    try {
      await api.saveMovieDetails(infoHash, movieDetails);
      updateTorrent(infoHash, { movieDetails });
    } catch (error) {
      console.error('保存电影详情失败:', error);
      throw error;
    }
  }, [updateTorrent]);

  return {
    loadTorrents,
    addNewTorrent,
    deleteTorrent,
    updateTorrentProgress,
    saveMovieDetails
  };
};

// 搜索相关的异步操作hooks
export const useSearchActions = () => {
  const {
    setSearchResults,
    setSearching,
    setSearchQuery
  } = useSearch();

  // 搜索电影
  const searchMovie = useCallback(async (query) => {
    if (!query.trim()) {
      setSearchResults([]);
      return;
    }

    try {
      setSearching(true);
      setSearchQuery(query);
      const results = await api.getMovieInfo(query);
      setSearchResults(results ? [results] : []);
    } catch (error) {
      console.error('搜索失败:', error);
      setSearchResults([]);
    } finally {
      setSearching(false);
    }
  }, [setSearchResults, setSearching, setSearchQuery]);

  // 清除搜索结果
  const clearSearch = useCallback(() => {
    setSearchQuery('');
    setSearchResults([]);
  }, [setSearchQuery, setSearchResults]);

  return {
    searchMovie,
    clearSearch
  };
};

// 通用的错误处理hook
export const useErrorHandler = () => {
  const { setTorrentError } = useTorrents();

  const handleError = useCallback((error, context = '') => {
    console.error(`错误[${context}]:`, error);
    
    let errorMessage = '发生未知错误';
    if (error.message) {
      errorMessage = error.message;
    } else if (typeof error === 'string') {
      errorMessage = error;
    }

    setTorrentError(`${context ? context + ': ' : ''}${errorMessage}`);
  }, [setTorrentError]);

  const clearError = useCallback(() => {
    setTorrentError(null);
  }, [setTorrentError]);

  return {
    handleError,
    clearError
  };
};

// 播放器相关操作hooks
export const usePlayerActions = () => {
  const { setCurrentPlayingFile } = useAppStore();

  const playFile = useCallback((infoHash, fileIndex, fileName) => {
    setCurrentPlayingFile({
      infoHash,
      fileIndex,
      fileName
    });
  }, [setCurrentPlayingFile]);

  const stopPlaying = useCallback(() => {
    setCurrentPlayingFile(null);
  }, [setCurrentPlayingFile]);

  return {
    playFile,
    stopPlaying
  };
};