package db

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "modernc.org/sqlite"
)

// Migration 数据库迁移结构
type Migration struct {
	Version     int
	Description string
	SQL         string
}

// migrations 所有数据库迁移
var migrations = []Migration{
	{
		Version:     1,
		Description: "创建torrents表",
		SQL: `
			CREATE TABLE IF NOT EXISTS torrents (
				info_hash TEXT PRIMARY KEY,
				name TEXT NOT NULL,
				magnet_uri TEXT NOT NULL,
				added_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
				data_path TEXT,
				length INTEGER DEFAULT 0,
				files TEXT,
				downloaded INTEGER DEFAULT 0,
				progress REAL DEFAULT 0,
				state TEXT DEFAULT 'downloading',
				movie_details TEXT,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)
		`,
	},
	{
		Version:     2,
		Description: "添加索引",
		SQL: `
			CREATE INDEX IF NOT EXISTS idx_torrents_name ON torrents(name);
			CREATE INDEX IF NOT EXISTS idx_torrents_added_at ON torrents(added_at);
			CREATE INDEX IF NOT EXISTS idx_torrents_state ON torrents(state);
			CREATE INDEX IF NOT EXISTS idx_torrents_progress ON torrents(progress);
		`,
	},
	{
		Version:     3,
		Description: "创建迁移版本表",
		SQL: `
			CREATE TABLE IF NOT EXISTS schema_migrations (
				version INTEGER PRIMARY KEY,
				applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)
		`,
	},
	{
		Version:     4,
		Description: "添加性能优化设置",
		SQL: `
			PRAGMA journal_mode=WAL;
			PRAGMA synchronous=NORMAL;
			PRAGMA cache_size=10000;
			PRAGMA temp_store=MEMORY;
		`,
	},
}

// DatabaseManager 数据库管理器
type DatabaseManager struct {
	db             *sql.DB
	maxConnections int
	connTimeout    time.Duration
}

// NewDatabaseManager 创建数据库管理器
func NewDatabaseManager(dbPath string, maxConnections int, connTimeout time.Duration) (*DatabaseManager, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}

	// 设置连接池参数
	db.SetMaxOpenConns(maxConnections)
	db.SetMaxIdleConns(maxConnections / 2)
	db.SetConnMaxLifetime(connTimeout)

	manager := &DatabaseManager{
		db:             db,
		maxConnections: maxConnections,
		connTimeout:    connTimeout,
	}

	// 执行数据库迁移
	if err := manager.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("数据库迁移失败: %w", err)
	}

	return manager, nil
}

// migrate 执行数据库迁移
func (dm *DatabaseManager) migrate() error {
	log.Println("开始数据库迁移...")

	// 首先创建迁移表（如果不存在）
	if err := dm.createMigrationTable(); err != nil {
		return fmt.Errorf("创建迁移表失败: %w", err)
	}

	// 获取已应用的迁移版本
	appliedVersions, err := dm.getAppliedVersions()
	if err != nil {
		return fmt.Errorf("获取迁移版本失败: %w", err)
	}

	// 执行未应用的迁移
	for _, migration := range migrations {
		if !contains(appliedVersions, migration.Version) {
			log.Printf("应用迁移 v%d: %s", migration.Version, migration.Description)
			
			if err := dm.applyMigration(migration); err != nil {
				return fmt.Errorf("应用迁移 v%d 失败: %w", migration.Version, err)
			}
			
			log.Printf("迁移 v%d 应用成功", migration.Version)
		}
	}

	log.Println("数据库迁移完成")
	return nil
}

// createMigrationTable 创建迁移表
func (dm *DatabaseManager) createMigrationTable() error {
	_, err := dm.db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	return err
}

// getAppliedVersions 获取已应用的迁移版本
func (dm *DatabaseManager) getAppliedVersions() ([]int, error) {
	rows, err := dm.db.Query("SELECT version FROM schema_migrations ORDER BY version")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []int
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		versions = append(versions, version)
	}

	return versions, rows.Err()
}

// applyMigration 应用单个迁移
func (dm *DatabaseManager) applyMigration(migration Migration) error {
	tx, err := dm.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 执行迁移SQL
	if _, err := tx.Exec(migration.SQL); err != nil {
		return fmt.Errorf("执行迁移SQL失败: %w", err)
	}

	// 记录迁移版本
	if _, err := tx.Exec(
		"INSERT INTO schema_migrations (version) VALUES (?)",
		migration.Version,
	); err != nil {
		return fmt.Errorf("记录迁移版本失败: %w", err)
	}

	return tx.Commit()
}

// GetDB 获取数据库连接
func (dm *DatabaseManager) GetDB() *sql.DB {
	return dm.db
}

// Close 关闭数据库连接
func (dm *DatabaseManager) Close() error {
	return dm.db.Close()
}

// Ping 检查数据库连接
func (dm *DatabaseManager) Ping() error {
	return dm.db.Ping()
}

// Stats 获取数据库连接池统计信息
func (dm *DatabaseManager) Stats() sql.DBStats {
	return dm.db.Stats()
}

// LogStats 记录数据库连接池统计信息
func (dm *DatabaseManager) LogStats() {
	stats := dm.Stats()
	log.Printf("数据库连接池统计: 打开连接=%d, 使用中=%d, 空闲=%d, 等待=%d",
		stats.OpenConnections,
		stats.InUse,
		stats.Idle,
		stats.WaitCount,
	)
}

// Optimize 优化数据库
func (dm *DatabaseManager) Optimize() error {
	log.Println("开始数据库优化...")
	
	optimizations := []string{
		"VACUUM",                          // 重新组织数据库文件
		"ANALYZE",                         // 更新查询计划器统计信息
		"PRAGMA optimize",                 // 自动优化
		"PRAGMA wal_checkpoint(TRUNCATE)", // WAL模式下清理日志
	}

	for _, sql := range optimizations {
		if _, err := dm.db.Exec(sql); err != nil {
			log.Printf("执行优化命令失败 '%s': %v", sql, err)
			// 继续执行其他优化，不中断
		}
	}

	log.Println("数据库优化完成")
	return nil
}

// 辅助函数

// contains 检查切片是否包含指定元素
func contains(slice []int, item int) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}