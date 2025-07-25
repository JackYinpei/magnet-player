# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

这是一个重构后的Torrent媒体播放器项目，支持磁力链接下载和流媒体播放。项目采用现代化的分层架构：

- **Backend**: Go服务，采用分层架构（Handler -> Service -> Repository）
- **Frontend**: Next.js应用，使用TypeScript和Zustand状态管理
- **Signaling**: WebRTC信令服务器，支持P2P功能

## 重构亮点

### 后端架构重构
1. **分层架构**: 清晰的Handler -> Service -> Repository分层
2. **依赖注入**: 统一的配置管理和依赖注入
3. **中间件系统**: 统一的错误处理、CORS、验证、日志中间件
4. **数据库优化**: 连接池、索引、迁移系统
5. **输入验证**: 全面的安全输入验证

### 前端架构重构
1. **TypeScript支持**: 完整的类型定义和类型安全
2. **状态管理**: 使用Zustand替代本地状态
3. **异步操作**: 统一的异步操作hooks
4. **错误处理**: 全局错误处理机制

## 开发命令

### 后端 (Backend)
```bash
cd backend
go run main_new.go              # 启动重构后的服务器
go run main.go                  # 启动原始服务器 (legacy)
go test ./...                   # 运行所有测试
go test ./validator/            # 运行验证器测试
go test ./service/              # 运行服务层测试
go mod tidy                     # 整理依赖
```

### 前端 (Frontend) 
```bash
cd frontend
npm run dev                     # 开发模式 (localhost:3000)
npm run build                   # 构建生产版本
npm run start                   # 启动生产服务
npm run lint                    # 代码检查
npm run type-check              # TypeScript类型检查
```

### 信令服务器 (Signaling)
```bash
cd signaling
go run cmd/signaling/main.go    # 启动信令服务器

cd signalingv2  
go run cmd/server/main.go       # 启动v2信令服务器
```

## 重构后的架构要点

### 后端分层架构

#### 1. 配置层 (config/)
- `config.go`: 统一配置管理，支持环境变量和默认值
- 类型安全的配置结构
- 生产/开发环境自动检测

#### 2. 中间件层 (middleware/)
- `cors.go`: 可配置的CORS处理
- `error.go`: 统一错误处理和恢复
- `logger.go`: 请求日志记录
- `validation.go`: 输入验证中间件

#### 3. 验证层 (validator/)
- `validator.go`: 磁力链接、文件路径、InfoHash验证
- 安全的输入验证，防止注入攻击
- 统一的验证错误格式

#### 4. 数据层 (db/)
- `migrations.go`: 数据库迁移和优化
- `torrent_store.go`: 优化的数据访问层
- 连接池管理和性能优化
- 读写锁优化并发访问

#### 5. 服务层 (service/)
- `torrent_service.go`: 种子业务逻辑
- `search_service.go`: 搜索业务逻辑
- 业务逻辑封装，与HTTP层解耦

#### 6. 处理层 (handlers/)
- `torrent_handler.go`: 种子相关HTTP处理
- `stream_handler.go`: 流媒体HTTP处理  
- `search_handler.go`: 搜索HTTP处理
- 只负责HTTP协议处理，业务逻辑委托给服务层

### 前端架构重构

#### 1. 类型系统 (types/)
- `index.ts`: 全局TypeScript类型定义
- 完整的接口定义和类型安全

#### 2. 状态管理 (lib/)
- `store.ts`: Zustand状态管理，支持持久化
- `actions.ts`: 异步操作hooks
- 分片式状态管理，性能优化

#### 3. API层 (lib/)
- `api.js`: 优化的API客户端，统一错误处理
- 类型安全的API调用

## API端点架构

### 重构后的端点
- `POST /magnet/api/magnet`: 添加磁力链接（增强验证）
- `GET /magnet/api/torrents`: 列出所有种子
- `GET /magnet/stream/{infoHash}/{fileName}`: 流媒体文件（安全验证）
- `GET /magnet/search?filename={name}`: 搜索电影（参数验证）
- `POST /magnet/api/movie-details/{infoHash}`: 保存电影详情
- `GET /magnet/api/get-movie-details`: 获取所有电影详情
- `POST /magnet/api/torrents/save-data/{infoHash}`: 保存种子数据

### 安全增强
- 输入验证中间件
- CORS配置可定制化
- 错误信息统一格式
- 请求日志记录
- 文件路径安全检查

## 技术栈升级

### 后端技术栈
- Go 1.23 (现代化Go特性)
- 分层架构模式
- 中间件模式
- 依赖注入
- 数据库连接池
- 类型安全的配置管理

### 前端技术栈  
- Next.js 15 + React 19
- TypeScript 5.8+ (完整类型支持)
- Zustand (轻量级状态管理)
- Radix UI + Tailwind CSS
- 异步操作hooks模式

### 数据库优化
- SQLite WAL模式
- 连接池管理
- 自动迁移系统
- 性能索引
- 查询优化

## 开发最佳实践

### 后端开发
1. 使用分层架构，保持职责分离
2. 所有用户输入必须通过验证层
3. 使用中间件处理横切关注点
4. 错误处理要统一和一致
5. 使用配置管理而不是硬编码

### 前端开发
1. 使用TypeScript，保持类型安全
2. 状态管理通过Zustand集中化
3. 异步操作使用专门的hooks
4. 组件保持单一职责原则
5. 错误边界和错误处理要完善

### 测试策略
- 单元测试：验证器、服务层逻辑
- 集成测试：API端点和数据库交互
- 前端测试：组件测试和状态管理测试

## 环境配置

### 必需的环境变量
```bash
# API密钥
JINA_API_KEY=your_jina_api_key
TMDB_API_KEY=your_tmdb_api_key
OPENAI_API_KEY=your_openai_api_key

# 服务器配置
SERVER_HOST=localhost
SERVER_PORT=8080
ENV=development

# 数据库配置  
DB_PATH=./data/torrents.db
DB_MAX_CONNECTIONS=10

# Torrent配置
TORRENT_DATA_DIR=./data
TORRENT_MAX_CONNECTIONS=50
```

### 开发环境启动步骤
1. 复制环境变量模板：`cp .env.example .env`
2. 填写必要的API密钥
3. 启动后端：`cd backend && go run main_new.go`
4. 启动前端：`cd frontend && npm run dev`
5. 访问：http://localhost:3000

## 重构收益

1. **代码质量**: 更好的可读性、可维护性、可测试性
2. **性能优化**: 数据库连接池、查询优化、状态管理优化
3. **安全性**: 输入验证、CORS配置、错误处理
4. **开发体验**: TypeScript类型安全、更好的错误提示
5. **可扩展性**: 分层架构便于功能扩展和修改