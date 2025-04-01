'use client';

import React, { useState, useEffect, useRef } from 'react';
import styles from './pplayer.module.css';

export default function PPlayer() {
  // 状态变量
  const [filePathInput, setFilePathInput] = useState('');
  const [isConnected, setIsConnected] = useState(false);
  const [status, setStatus] = useState('未连接');
  const [logs, setLogs] = useState([]);
  const [videoVisible, setVideoVisible] = useState(false);
  const [isPlaying, setIsPlaying] = useState(false);
  const [autoplayBlocked, setAutoplayBlocked] = useState(false);

  // refs
  const videoPlayerRef = useRef(null);
  const logContainerRef = useRef(null);
  const websocketRef = useRef(null);
  const peerConnectionRef = useRef(null);
  const dataChannelRef = useRef(null);
  const mediaSourceRef = useRef(null);
  const sourceBufferRef = useRef(null);
  const queueRef = useRef([]);
  const isBufferReadyRef = useRef(true);
  const videoReceiveBufferRef = useRef([]);
  const receivedSizeRef = useRef(0);
  const isReceivingFileRef = useRef(false);
  const currentSeekRef = useRef(0);
  const pendingIceCandidatesRef = useRef([]);
  const remoteDescriptionSetRef = useRef(false);

  // 常量
  const signalServer = 'wss://shiying.sh.cn:8090/ws';
  const BUFFER_AHEAD = 5 * 60; // 缓冲区保持5分钟
  const clientId = `consumer-${Date.now()}`; // 添加客户端唯一ID

  // 日志辅助函数
  const log = (message) => {
    const timestamp = new Date().toLocaleTimeString();
    setLogs(prevLogs => [...prevLogs, `[${timestamp}] ${message}`]);
    // 滚动到最新日志
    if (logContainerRef.current) {
      logContainerRef.current.scrollTop = logContainerRef.current.scrollHeight;
    }
  };

  // 更新状态显示
  const updateStatus = (message) => {
    setStatus(message);
  };

  // 等待MediaSource打开
  const waitForSourceOpen = (mediaSource) => {
    return new Promise((resolve) => {
      if (mediaSource.readyState === 'open') {
        resolve();
      } else {
        mediaSource.addEventListener('sourceopen', () => resolve());
      }
    });
  };

  // 处理缓冲区溢出
  const handleQuotaExceeded = (data) => {
    if (sourceBufferRef.current && sourceBufferRef.current.buffered.length > 0) {
      const currentTime = videoPlayerRef.current.currentTime;
      const bufferedStart = sourceBufferRef.current.buffered.start(0);

      if (currentTime - bufferedStart > 10) {
        const removeEnd = currentTime - 5;
        log(`清理缓冲区: ${bufferedStart.toFixed(2)}s 到 ${removeEnd.toFixed(2)}s`);
        
        // 检查sourceBuffer是否正在更新
        if (!sourceBufferRef.current.updating) {
          sourceBufferRef.current.remove(bufferedStart, removeEnd);
          // 在下一个更新周期处理队列中的数据
          queueRef.current.unshift(data);
        } else {
          // 如果正在更新，添加事件监听器在更新结束后尝试清理
          const onUpdateEnd = () => {
            sourceBufferRef.current.removeEventListener('updateend', onUpdateEnd);
            handleQuotaExceeded(data);
          };
          sourceBufferRef.current.addEventListener('updateend', onUpdateEnd);
        }
      } else {
        // 如果没有足够的缓冲区可清理，尝试移除整个缓冲区并重新开始
        if (!sourceBufferRef.current.updating) {
          log('缓冲区过小无法清理，尝试移除所有缓冲区');
          sourceBufferRef.current.remove(bufferedStart, sourceBufferRef.current.buffered.end(0));
          queueRef.current.unshift(data);
        }
      }
    }
  };

  // 处理数据队列
  const processQueue = () => {
    if (!isBufferReadyRef.current || 
        queueRef.current.length === 0 || 
        !sourceBufferRef.current || 
        !mediaSourceRef.current || 
        mediaSourceRef.current.readyState !== 'open') {
      return;
    }

    isBufferReadyRef.current = false;
    const data = queueRef.current.shift();

    try {
      // 检查sourceBuffer是否可以接收数据
      if (!sourceBufferRef.current.updating) {
        log(`添加 ${data.byteLength} 字节到缓冲区`);
        sourceBufferRef.current.appendBuffer(new Uint8Array(data));
        
        // 当缓冲区有数据时尝试播放
        if (sourceBufferRef.current.buffered.length > 0 && 
            videoPlayerRef.current && 
            videoPlayerRef.current.paused) {
          // 记录缓冲区状态
          const start = sourceBufferRef.current.buffered.start(0);
          const end = sourceBufferRef.current.buffered.end(0);
          log(`缓冲区状态: ${start.toFixed(2)}s - ${end.toFixed(2)}s (${(end-start).toFixed(2)}s)`);
          
          // 确保视频元素有src属性
          if (!videoPlayerRef.current.src) {
            log('视频元素src为空，重新设置...');
            if (mediaSourceRef.current) {
              const objectUrl = URL.createObjectURL(mediaSourceRef.current);
              videoPlayerRef.current.src = objectUrl;
              log(`重新设置视频src: ${objectUrl}`);
              videoPlayerRef.current.load();
            } else {
              log('错误: MediaSource未就绪，无法设置视频src');
              return;
            }
          }
          
          // 检查src是否有效
          if (!videoPlayerRef.current.src || videoPlayerRef.current.src === '') {
            log('警告: 视频src仍然为空，无法播放');
            return;
          }
          
          // 当有足够数据时尝试播放
          log('缓冲区有数据，尝试播放视频...');
          const playPromise = videoPlayerRef.current.play();
          if (playPromise !== undefined) {
            playPromise
              .then(() => {
                setIsPlaying(true);
                setAutoplayBlocked(false);
                log('视频开始播放');
              })
              .catch(error => {
                setAutoplayBlocked(true);
                log(`自动播放被阻止: ${error.message}`);
              });
          }
        }
      } else {
        // 如果sourceBuffer正在更新，重新放回队列
        queueRef.current.unshift(data);
        setTimeout(processQueue, 100);
      }
    } catch (error) {
      log(`添加数据到缓冲区时出错: ${error.message}`);
      
      // 重新放回队列
      queueRef.current.unshift(data);
      
      if (error.name === 'QuotaExceededError') {
        handleQuotaExceeded(data);
      } else {
        // 其他错误，等待一段时间后重试
        setTimeout(() => {
          isBufferReadyRef.current = true;
          processQueue();
        }, 500);
      }
    }
  };

  // 合并多个ArrayBuffer为一个
  const combineArrayBuffers = (buffers) => {
    let totalLength = 0;
    buffers.forEach(buffer => {
      totalLength += buffer.byteLength;
    });
    
    const result = new Uint8Array(totalLength);
    let offset = 0;
    
    buffers.forEach(buffer => {
      result.set(new Uint8Array(buffer), offset);
      offset += buffer.byteLength;
    });
    
    return result.buffer;
  };

  // 初始化WebRTC连接
  const initWebRTC = () => {
    log('开始初始化WebRTC连接...');
    
    const config = {
      iceServers: [
        { urls: 'stun:stun.l.google.com:19302' }
      ]
    };
    
    const pc = new RTCPeerConnection(config);
    peerConnectionRef.current = pc;
    
    // 监听ICE候选
    pc.onicecandidate = event => {
      if (event.candidate) {
        log('发送ICE候选到生产者...');
        const message = {
          type: 'ice-candidate',
          data: {
            ...event.candidate.toJSON(),
            clientId: clientId
          }
        };
        
        if (websocketRef.current && websocketRef.current.readyState === WebSocket.OPEN) {
          websocketRef.current.send(JSON.stringify(message));
        }
      }
    };
    
    // 监听连接状态
    pc.onconnectionstatechange = () => {
      log(`连接状态: ${pc.connectionState}`);
      if (pc.connectionState === 'connected') {
        // 连接建立后立即初始化MediaSource
        log('WebRTC连接已建立，初始化MediaSource...');
        setupMediaSource().catch(err => {
          log(`MediaSource初始化失败: ${err.message}`);
        });
      }
    };
    
    // 监听ICE连接状态
    pc.oniceconnectionstatechange = () => {
      log(`ICE连接状态: ${pc.iceConnectionState}`);
    };
    
    // 监听数据通道
    pc.ondatachannel = event => {
      log('接收到数据通道...');
      const channel = event.channel;
      dataChannelRef.current = channel;
      setupDataChannel(channel);
    };
    
    log('WebRTC初始化完成');
    
    // 发送连接请求
    sendConnectRequest();
  };
  
  // 发送连接请求到生产者
  const sendConnectRequest = () => {
    if (websocketRef.current && websocketRef.current.readyState === WebSocket.OPEN) {
      log('向信令服务器发送连接就绪消息');
      const message = {
        type: 'connect',
        data: {
          clientId: clientId
        }
      };
      websocketRef.current.send(JSON.stringify(message));
    } else {
      log('WebSocket未连接，无法发送连接请求');
    }
  };
  
  // 处理信令消息
  const handleSignalingMessage = (event) => {
    try {
      const message = JSON.parse(event.data);
      log(`收到信令: ${message.type}`);
      
      switch (message.type) {
        case 'offer':
          handleOffer(message.data);
          break;
          
        case 'ice-candidate':
          handleIceCandidate(message.data);
          break;
          
        default:
          log(`未知消息类型: ${message.type}`);
      }
    } catch (error) {
      log(`处理信令消息出错: ${error.message}`);
    }
  };
  
  // 处理Offer
  const handleOffer = (data) => {
    log('收到生产者SDP报价...');
    
    let offerData = data;
    if (typeof data === 'string') {
      try {
        offerData = JSON.parse(data);
      } catch (error) {
        log(`解析SDP数据出错: ${error.message}`);
        return;
      }
    }
    
    log('解析的SDP数据: ' + JSON.stringify(offerData));
    
    const rtcSessionDescription = new RTCSessionDescription({
      type: 'offer',
      sdp: offerData.sdp
    });
    
    // 设置远程描述
    log('正在设置远程SDP...');
    peerConnectionRef.current.setRemoteDescription(rtcSessionDescription)
      .then(() => {
        // 创建应答
        log('创建SDP应答...');
        return peerConnectionRef.current.createAnswer();
      })
      .then(answer => {
        // 设置本地描述
        log('设置本地SDP应答...');
        return peerConnectionRef.current.setLocalDescription(answer);
      })
      .then(() => {
        // 发送应答到生产者
        log('向生产者发送SDP应答...');
        const answerMessage = {
          type: 'answer',
          data: {
            sdp: peerConnectionRef.current.localDescription.sdp,
            type: 'answer',
            clientId: clientId
          }
        };
        
        if (websocketRef.current && websocketRef.current.readyState === WebSocket.OPEN) {
          websocketRef.current.send(JSON.stringify(answerMessage));
        }
      })
      .catch(error => {
        log(`处理offer出错: ${error.message}`);
      });
  };
  
  // 处理ICE候选
  const handleIceCandidate = (data) => {
    log('收到生产者ICE候选...');
    
    let iceData = data;
    if (typeof data === 'string') {
      try {
        iceData = JSON.parse(data);
      } catch (error) {
        log(`解析ICE数据出错: ${error.message}`);
        return;
      }
    }
    
    log('解析的ICE数据: ' + JSON.stringify(iceData));
    
    // 创建并添加ICE候选
    const candidate = new RTCIceCandidate({
      candidate: iceData.candidate,
      sdpMid: iceData.sdpMid || '',
      sdpMLineIndex: iceData.sdpMLineIndex || 0
    });
    
    peerConnectionRef.current.addIceCandidate(candidate)
      .catch(error => {
        log(`添加ICE候选出错: ${error.message}`);
      });
  };
  
  // 设置数据通道
  const setupDataChannel = (channel) => {
    channel.binaryType = 'arraybuffer';
    
    channel.onopen = () => {
      log('数据通道已打开，可以开始传输数据');
      updateStatus('已连接 - 数据通道已就绪');
      setIsConnected(true);
      
      // 初始化MediaSource - 确保在数据通道打开后直接初始化
      setupMediaSource();
    };
    
    channel.onclose = () => {
      log('数据通道已关闭');
      updateStatus('数据通道已关闭');
      setIsConnected(false);
      
      // 如果MediaSource仍然打开，结束流
      if (mediaSourceRef.current && mediaSourceRef.current.readyState === 'open') {
        try {
          mediaSourceRef.current.endOfStream();
        } catch (e) {
          log(`关闭MediaSource时出错: ${e.message}`);
        }
      }
    };
    
    channel.onerror = (error) => {
      log(`数据通道错误: ${error.message || '未知错误'}`);
    };
    
    channel.onmessage = (event) => {
      if (typeof event.data === 'string') {
        try {
          const message = JSON.parse(event.data);
          if (message.type === 'file-info') {
            log(`接收文件信息: ${message.name}, 大小: ${message.size} 字节`);
            isReceivingFileRef.current = true;
            receivedSizeRef.current = 0;
            videoReceiveBufferRef.current = [];
            currentSeekRef.current = 0; // 重置播放位置
            
            // 重置MediaSource状态
            if (mediaSourceRef.current && mediaSourceRef.current.readyState === 'open') {
              try {
                resetPlayer();
                // 重新初始化MediaSource
                setupMediaSource();
              } catch (e) {
                log(`重置MediaSource失败: ${e.message}`);
              }
            }
            return;
          }
        } catch (e) {
          // 如果不是JSON，可能是普通文本消息
          log(`收到消息: ${event.data}`);
        }
        return;
      }
      
      // 处理二进制数据
      if (event.data instanceof ArrayBuffer) {
        if (isReceivingFileRef.current) {
          receivedSizeRef.current += event.data.byteLength;
          
          // 立即尝试处理接收到的视频数据
          if (mediaSourceRef.current && mediaSourceRef.current.readyState === 'open' && sourceBufferRef.current) {
            // 将接收到的数据直接加入队列
            queueRef.current.push(event.data);
            
            // 如果这是初始数据块，确保视频src已经设置
            if (receivedSizeRef.current <= event.data.byteLength) {
              log(`已接收第一个数据块，大小: ${event.data.byteLength} 字节`);
              
              // 确保视频元素设置了src属性
              if (!videoPlayerRef.current.src) {
                log('检测到视频src为空，重新设置...');
                if (mediaSourceRef.current) {
                  const objectUrl = URL.createObjectURL(mediaSourceRef.current);
                  videoPlayerRef.current.src = objectUrl;
                  log(`设置视频src: ${objectUrl}`);
                  videoPlayerRef.current.load();
                } else {
                  log('警告: MediaSource未就绪，无法设置视频src');
                  // 尝试重新初始化MediaSource
                  setupMediaSource().catch(err => {
                    log(`MediaSource初始化失败: ${err.message}`);
                  });
                }
              } else {
                log(`确认视频src已设置: ${videoPlayerRef.current.src}`);
              }
            }
            
            // 如果缓冲区准备好，立即处理队列
            if (isBufferReadyRef.current) {
              processQueue();
            }
            
            // 检查和记录接收进度
            if (receivedSizeRef.current % (1024 * 1024) < event.data.byteLength) {
              log(`已接收 ${(receivedSizeRef.current / (1024 * 1024)).toFixed(2)} MB 的视频数据`);
            }
          } else {
            // 如果MediaSource还没准备好，先收集数据
            videoReceiveBufferRef.current.push(event.data);
            
            // 如果收集了一定量的数据，尝试初始化MediaSource
            if (videoReceiveBufferRef.current.length === 1) {
              log(`已收集数据，等待MediaSource就绪...`);
              if (!mediaSourceRef.current) {
                setupMediaSource();
              }
            }
          }
        }
      }
    };
  };
  
  // 添加ICE候选，如果远程描述未设置则缓存
  const addIceCandidate = (candidate) => {
    if (peerConnectionRef.current) {
      if (remoteDescriptionSetRef.current) {
        log('添加ICE候选...');
        peerConnectionRef.current.addIceCandidate(candidate)
          .catch(error => {
            log(`添加ICE候选错误: ${error.message}`);
          });
      } else {
        log('远程描述未设置，缓存ICE候选');
        pendingIceCandidatesRef.current.push(candidate);
      }
    }
  };
  
  // 添加所有缓存的ICE候选
  const addPendingIceCandidates = () => {
    if (pendingIceCandidatesRef.current.length > 0) {
      log(`添加 ${pendingIceCandidatesRef.current.length} 个缓存的ICE候选...`);
      pendingIceCandidatesRef.current.forEach(candidate => {
        peerConnectionRef.current.addIceCandidate(candidate)
          .catch(error => {
            log(`添加缓存的ICE候选错误: ${error.message}`);
          });
      });
      pendingIceCandidatesRef.current = [];
    }
  };
  
  // 连接到信令服务器
  const connectToSignalServer = () => {
    if (websocketRef.current && websocketRef.current.readyState === WebSocket.OPEN) {
      websocketRef.current.close();
    }

    updateStatus('正在连接到信令服务器...');
    log('连接到信令服务器...');

    // 创建WebSocket连接，带上ID和类型
    websocketRef.current = new WebSocket(`${signalServer}?id=${clientId}&type=consumer`);
    
    websocketRef.current.onopen = () => {
      log('已连接到信令服务器');
      updateStatus('已连接到信令服务器 - 等待生产者连接');
      
      // 初始化WebRTC
      initWebRTC();

      // 告知生产者有新的消费者连接，等待生产者发起offer
      const connectMessage = {
        type: 'connect',
        data: {
          clientId: clientId,
          message: '消费者连接就绪，等待生产者发起连接'
        }
      };

      if (websocketRef.current && websocketRef.current.readyState === WebSocket.OPEN) {
        websocketRef.current.send(JSON.stringify(connectMessage));
        log('发送连接就绪消息到信令服务器');
      }
    };
    
    websocketRef.current.onmessage = handleSignalingMessage;
    
    websocketRef.current.onclose = () => {
      log('与信令服务器的连接已关闭');
      updateStatus('未连接 - 信令服务器连接已关闭');
      setIsConnected(false);
    };
    
    websocketRef.current.onerror = (error) => {
      log(`信令服务器错误: ${error.message || '未知错误'}`);
      updateStatus('未连接 - 信令服务器错误');
    };
  };
  
  // 获取指定文件
  const requestFile = () => {
    if (!dataChannelRef.current || dataChannelRef.current.readyState !== 'open') {
      log('数据通道未打开，无法请求文件');
      return;
    }
    
    if (!filePathInput.trim()) {
      log('请输入要请求的文件路径');
      return;
    }
    
    // 首先重置播放器
    resetPlayer();
    
    // 确保MediaSource已设置好
    const setupAndRequest = async () => {
      try {
        log('开始设置MediaSource...');
        await setupMediaSource();
        
        // 确认视频元素已设置src
        if (!videoPlayerRef.current.src && mediaSourceRef.current) {
          const objectUrl = URL.createObjectURL(mediaSourceRef.current);
          videoPlayerRef.current.src = objectUrl;
          log(`设置视频src: ${objectUrl}`);
          videoPlayerRef.current.load();
        }
        
        // 确认src已设置
        if (!videoPlayerRef.current.src) {
          throw new Error('无法设置视频src属性');
        }
        
        // MediaSource准备就绪，发送请求
        log(`请求文件: ${filePathInput}`);
        dataChannelRef.current.send(JSON.stringify({
          type: 'request-file',
          path: filePathInput
        }));
      } catch (error) {
        log(`设置媒体源失败: ${error.message}`);
      }
    };
    
    setupAndRequest();
  };
  
  // 初始化MediaSource
  const setupMediaSource = async () => {
    try {
      log('初始化MediaSource...');
      
      // 如果已经存在并且开启状态，直接使用
      if (mediaSourceRef.current && mediaSourceRef.current.readyState === 'open') {
        log('MediaSource已处于打开状态，跳过初始化');
        return;
      }
      
      // 如果已经存在但不是open状态，先清理
      if (mediaSourceRef.current) {
        log('释放旧的MediaSource...');
        resetPlayer();
      }
      
      // 创建MediaSource
      mediaSourceRef.current = new MediaSource();
      setVideoVisible(true);
      
      // 立即连接到视频元素
      if (videoPlayerRef.current) {
        const objectUrl = URL.createObjectURL(mediaSourceRef.current);
        videoPlayerRef.current.src = objectUrl;
        log(`设置视频src: ${objectUrl}`);
        videoPlayerRef.current.load();
      } else {
        log('错误: 视频元素尚未准备好');
        throw new Error('视频元素未准备好');
      }
      
      // 等待MediaSource打开
      log('等待MediaSource打开...');
      await waitForSourceOpen(mediaSourceRef.current);
      log('MediaSource已打开，添加SourceBuffer...');
      
      // 支持的MIME类型列表
      const mimeTypes = [
        'video/mp4; codecs="avc1.42E01E, mp4a.40.2"',
        'video/mp4; codecs="avc1.4D401E, mp4a.40.2"',
        'video/mp4; codecs="avc1.64001E, mp4a.40.2"',
        'video/mp4; codecs="avc1.640029, mp4a.40.2"',
        'video/mp4; codecs="avc1.42E01F, mp4a.40.2"',
        'video/mp4; codecs="avc1.42E01E"',
        'video/mp4; codecs="avc1.4D401E"',
        'video/mp4; codecs="avc1.64001E"',
        'video/mp4; codecs="avc1.640029"',
        'video/mp4; codecs="avc1.42E01F"',
        'video/mp4'
      ];
      
      // 尝试添加SourceBuffer
      let sourceBuffer = null;
      
      for (const mimeType of mimeTypes) {
        if (MediaSource.isTypeSupported(mimeType)) {
          try {
            sourceBuffer = mediaSourceRef.current.addSourceBuffer(mimeType);
            log(`已添加SourceBuffer，MIME类型: ${mimeType}`);
            break;
          } catch (e) {
            log(`尝试MIME类型 ${mimeType} 失败: ${e.message}`);
          }
        }
      }
      
      if (!sourceBuffer) {
        throw new Error('未找到支持的MIME类型');
      }
      
      // 配置SourceBuffer
      sourceBufferRef.current = sourceBuffer;
      sourceBuffer.mode = 'segments';
      
      // 添加事件监听器
      sourceBuffer.addEventListener('updateend', () => {
        isBufferReadyRef.current = true;
        
        // 处理队列中剩余的数据
        processQueue();
        
        // 记录缓冲区状态
        if (sourceBuffer.buffered.length > 0) {
          const start = sourceBuffer.buffered.start(0);
          const end = sourceBuffer.buffered.end(0);
          log(`缓冲区状态: ${start.toFixed(2)}s - ${end.toFixed(2)}s (${(end-start).toFixed(2)}s)`);
          
          // 如果视频暂停了但有足够的缓冲，尝试播放
          if (videoPlayerRef.current && videoPlayerRef.current.paused && 
              end - videoPlayerRef.current.currentTime > 2) {
            playVideo();
          }
        }
      });
      
      sourceBuffer.addEventListener('error', (e) => {
        log(`SourceBuffer错误: ${e.message || '未知错误'}`);
      });
      
      // 添加视频播放事件监听
      videoPlayerRef.current.addEventListener('timeupdate', () => {
        // 检查并清理过时的缓冲区
        if (sourceBufferRef.current && sourceBufferRef.current.buffered.length > 0) {
          const currentTime = videoPlayerRef.current.currentTime;
          const bufferedStart = sourceBufferRef.current.buffered.start(0);
          const bufferedEnd = sourceBufferRef.current.buffered.end(0);
          
          // 记录当前播放进度和缓冲状态
          if (currentTime % 5 < 0.1) {  // 每5秒记录一次
            log(`播放进度: ${currentTime.toFixed(2)}s，缓冲区: ${bufferedStart.toFixed(2)}s - ${bufferedEnd.toFixed(2)}s`);
          }
          
          // 智能清理：播放位置超过缓冲区开始10秒，且缓冲区总长度超过30秒
          if (currentTime - bufferedStart > 10 && bufferedEnd - bufferedStart > 30) {
            if (!sourceBufferRef.current.updating) {
              const removeEnd = currentTime - 5;
              log(`自动清理缓冲区: ${bufferedStart.toFixed(2)}s 到 ${removeEnd.toFixed(2)}s`);
              sourceBufferRef.current.remove(bufferedStart, removeEnd);
            }
          }
        }
      });
      
      videoPlayerRef.current.addEventListener('playing', () => {
        setIsPlaying(true);
        setAutoplayBlocked(false);
        log('视频正在播放');
      });
      
      videoPlayerRef.current.addEventListener('pause', () => {
        setIsPlaying(false);
        log('视频已暂停');
      });
      
      videoPlayerRef.current.addEventListener('error', (e) => {
        log(`视频错误: ${e.target.error?.message || '未知错误'}`);
      });
      
      videoPlayerRef.current.addEventListener('stalled', () => {
        log('视频播放已停滞');
      });
      
      videoPlayerRef.current.addEventListener('waiting', () => {
        log('视频正在等待更多数据...');
      });
      
      // 如果已经收集了数据，现在处理它们
      if (videoReceiveBufferRef.current.length > 0) {
        log(`处理已收集的 ${videoReceiveBufferRef.current.length} 个数据块...`);
        
        // 将所有收集的数据添加到队列
        videoReceiveBufferRef.current.forEach(buffer => {
          queueRef.current.push(buffer);
        });
        
        // 清空临时缓冲区
        videoReceiveBufferRef.current = [];
        
        // 处理队列
        if (isBufferReadyRef.current) {
          processQueue();
        }
      }
    } catch (error) {
      log(`设置MediaSource失败: ${error.message}`);
    }
  };
  
  // 播放视频 (用于自动播放被阻止时的手动播放)
  const playVideo = () => {
    if (videoPlayerRef.current) {
      // 确认src已设置
      if (!videoPlayerRef.current.src || videoPlayerRef.current.src === '') {
        log('无法播放视频，src属性为空');
        
        // 尝试重新设置src
        if (mediaSourceRef.current) {
          const objectUrl = URL.createObjectURL(mediaSourceRef.current);
          videoPlayerRef.current.src = objectUrl;
          log(`重新设置视频src: ${objectUrl}`);
          videoPlayerRef.current.load();
        } else {
          log('无法播放：MediaSource未初始化');
          return;
        }
      }
      
      log('尝试播放视频...');
      videoPlayerRef.current.play()
        .then(() => {
          setIsPlaying(true);
          setAutoplayBlocked(false);
          log('视频开始播放');
        })
        .catch(err => {
          log(`播放失败: ${err.message}`);
        });
    } else {
      log('视频元素未找到，无法播放');
    }
  };
  
  // 重置播放器状态
  const resetPlayer = () => {
    log('重置播放器状态...');
    
    // 清理MediaSource相关资源
    if (mediaSourceRef.current) {
      try {
        // 先记录src
        const currentSrc = videoPlayerRef.current?.src;
        
        if (mediaSourceRef.current.readyState === 'open') {
          mediaSourceRef.current.endOfStream();
          log('MediaSource流已结束');
        }
        
        // 移除SourceBuffer
        if (sourceBufferRef.current && mediaSourceRef.current.readyState !== 'closed') {
          try {
            mediaSourceRef.current.removeSourceBuffer(sourceBufferRef.current);
            log('已移除SourceBuffer');
          } catch (e) {
            log(`移除SourceBuffer时出错: ${e.message}`);
          }
        }
        
        // 释放URL对象
        if (currentSrc && currentSrc.startsWith('blob:')) {
          URL.revokeObjectURL(currentSrc);
          log(`已释放Blob URL: ${currentSrc}`);
        }
        
        if (videoPlayerRef.current) {
          videoPlayerRef.current.src = '';
          videoPlayerRef.current.load();
        }
      } catch (e) {
        log(`关闭MediaSource时出错: ${e.message}`);
      }
    } else {
      log('没有活动的MediaSource需要重置');
    }
    
    // 重置引用
    mediaSourceRef.current = null;
    sourceBufferRef.current = null;
    queueRef.current = [];
    isBufferReadyRef.current = true;
    videoReceiveBufferRef.current = [];
    receivedSizeRef.current = 0;
    isReceivingFileRef.current = false;
    currentSeekRef.current = 0;
    
    // 重置状态
    setIsPlaying(false);
    setAutoplayBlocked(false);
  };
  
  // 清理资源
  useEffect(() => {
    return () => {
      // 清理WebRTC
      if (peerConnectionRef.current) {
        peerConnectionRef.current.close();
        peerConnectionRef.current = null;
      }
      
      if (dataChannelRef.current) {
        dataChannelRef.current.close();
        dataChannelRef.current = null;
      }
      
      // 清理WebSocket
      if (websocketRef.current) {
        websocketRef.current.close();
        websocketRef.current = null;
      }
      
      // 清理MediaSource
      resetPlayer();
    };
  }, []);
  
  return (
    <div className={styles.container}>
      <h1 className={styles.title}>P2P视频播放器</h1>
      
      <div className={styles.status}>
        <span>状态: {status}</span>
      </div>
      
      <div className={styles.controls}>
        <input
          type="text"
          placeholder="输入文件路径"
          value={filePathInput}
          onChange={(e) => setFilePathInput(e.target.value)}
          className={styles.input}
        />
        
        <button
          onClick={requestFile}
          disabled={!isConnected}
          className={styles.button}
        >
          请求文件
        </button>
        
        <button
          onClick={connectToSignalServer}
          disabled={isConnected}
          className={styles.button}
        >
          连接到服务器
        </button>
      </div>
      
      <div className={styles.examples}>
        <p>示例路径: (相对于 /root/magnet-player/backend/data):</p>
        <ul>
          <li>video.mp4</li>
          <li>movies/sample.mp4</li>
          <li>或绝对路径：/root/magnet-player/backend/data/video.mp4</li>
        </ul>
      </div>
      
      {/* 视频播放器区域 */}
      <div className={styles.videoContainer}>
        <video
          ref={videoPlayerRef}
          className={styles.videoPlayer}
          controls
          style={{ display: videoVisible ? 'block' : 'none' }}
        />
        
        {/* 自动播放被阻止时显示播放按钮覆盖层 */}
        {autoplayBlocked && videoVisible && (
          <div className={styles.playOverlay} onClick={playVideo}>
            <div className={styles.playButton}>
              <span>点击播放</span>
            </div>
          </div>
        )}
      </div>
      
      <div className={styles.logContainer} ref={logContainerRef}>
        <h3>连接日志</h3>
        <pre className={styles.log}>
          {logs.join('\n')}
        </pre>
      </div>
    </div>
  );
}