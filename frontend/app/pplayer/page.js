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
        sourceBufferRef.current.remove(bufferedStart, removeEnd);
      }
      queueRef.current.unshift(data);
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
      sourceBufferRef.current.appendBuffer(new Uint8Array(data));
      if (!isPlaying && videoPlayerRef.current.paused) {
        // 当有足够数据时尝试播放
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
    } catch (error) {
      log(`添加数据到缓冲区时出错: ${error.message}`);
      isBufferReadyRef.current = true;
      if (error.name === 'QuotaExceededError') {
        handleQuotaExceeded(data);
      }
    }
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
      
      // 初始化MediaSource
      setupMediaSource();
    };
    
    channel.onclose = () => {
      log('数据通道已关闭');
      updateStatus('数据通道已关闭');
      setIsConnected(false);
    };
    
    channel.onerror = (error) => {
      log(`数据通道错误: ${error.message || '未知错误'}`);
    };
    
    channel.onmessage = (event) => {
      if (typeof event.data === 'string') {
        const message = JSON.parse(event.data);
        if (message.type === 'file-info') {
          log(`接收文件信息: ${message.name}, 大小: ${message.size} 字节`);
          isReceivingFileRef.current = true;
          receivedSizeRef.current = 0;
          videoReceiveBufferRef.current = [];
          return;
        }
        // 处理其他文本消息
        log(`收到消息: ${event.data}`);
        return;
      }
      
      // 处理二进制数据
      if (event.data instanceof ArrayBuffer) {
        if (isReceivingFileRef.current) {
          receivedSizeRef.current += event.data.byteLength;
          
          // 添加到视频处理队列
          queueRef.current.push(event.data);
          if (isBufferReadyRef.current) {
            processQueue();
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
    
    // 清理旧的视频数据
    resetPlayer();
    
    // 请求文件
    log(`请求文件: ${filePathInput}`);
    dataChannelRef.current.send(filePathInput);
  };
  
  // 初始化MediaSource
  const setupMediaSource = async () => {
    try {
      // 创建MediaSource
      mediaSourceRef.current = new MediaSource();
      setVideoVisible(true);
      
      // 连接到视频元素
      videoPlayerRef.current.src = URL.createObjectURL(mediaSourceRef.current);
      videoPlayerRef.current.load();
      
      // 等待MediaSource打开
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
        processQueue();
        
        // 记录缓冲区状态
        if (sourceBuffer.buffered.length > 0) {
          const start = sourceBuffer.buffered.start(0);
          const end = sourceBuffer.buffered.end(0);
          log(`缓冲区状态: ${start.toFixed(2)}s - ${end.toFixed(2)}s (${(end-start).toFixed(2)}s)`);
        }
      });
      
      // 添加视频播放事件监听
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
      
    } catch (error) {
      log(`设置MediaSource失败: ${error.message}`);
    }
  };
  
  // 播放视频 (用于自动播放被阻止时的手动播放)
  const playVideo = () => {
    if (videoPlayerRef.current) {
      videoPlayerRef.current.play()
        .then(() => {
          setIsPlaying(true);
          setAutoplayBlocked(false);
        })
        .catch(err => {
          log(`播放失败: ${err.message}`);
        });
    }
  };
  
  // 重置播放器状态
  const resetPlayer = () => {
    // 清理MediaSource相关资源
    if (mediaSourceRef.current) {
      try {
        if (mediaSourceRef.current.readyState === 'open') {
          mediaSourceRef.current.endOfStream();
        }
      } catch (e) {
        log(`关闭MediaSource时出错: ${e.message}`);
      }
      
      // 移除SourceBuffer
      if (sourceBufferRef.current && mediaSourceRef.current.readyState !== 'closed') {
        try {
          mediaSourceRef.current.removeSourceBuffer(sourceBufferRef.current);
        } catch (e) {
          log(`移除SourceBuffer时出错: ${e.message}`);
        }
      }
      
      // 释放URL对象
      if (videoPlayerRef.current && videoPlayerRef.current.src) {
        URL.revokeObjectURL(videoPlayerRef.current.src);
        videoPlayerRef.current.src = '';
        videoPlayerRef.current.load();
      }
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