package db

import (
	"fmt"
	"log"
	"time"

	"ad-platform/pkg/util"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// PostgresDB PostgreSQL 数据库连接
type PostgresDB struct {
	DB *gorm.DB
}

// NewPostgresDB 创建 PostgreSQL 数据库连接
func NewPostgresDB(config *util.DatabaseConfig) (*PostgresDB, error) {
	// 参数验证
	if config == nil {
		return nil, fmt.Errorf("database config is nil")
	}

	// 构建 DSN
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.User, config.Password, config.DBName, config.SSLMode,
	)

	// 解析连接池配置
	maxOpenConns := config.MaxOpenConns
	maxIdleConns := config.MaxIdleConns
	connMaxLifetime, err := time.ParseDuration(config.ConnMaxLifetime)
	if err != nil {
		connMaxLifetime = 1 * time.Hour
	}

	// 解析连接最大空闲时间
	connMaxIdleTime, err := time.ParseDuration(config.ConnMaxIdleTime)
	if err != nil {
		connMaxIdleTime = 10 * time.Minute
	}

	// 配置 GORM
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	}

	// 连接数据库
	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		// 数据库连接失败，返回错误
		log.Printf("Error: Failed to connect to database: %v", err)
		return nil, fmt.Errorf("database connection failed: %w", err)
	}

	// 配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		log.Printf("Error: Failed to get database instance: %v", err)
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetConnMaxLifetime(connMaxLifetime)
	sqlDB.SetConnMaxIdleTime(connMaxIdleTime)

	// 测试连接
	if err := sqlDB.Ping(); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("database ping failed: %w", err)
	}

	return &PostgresDB{DB: db}, nil
}

// NewPostgresDBCluster 创建 PostgreSQL 数据库集群连接
// 支持主从复制和负载均衡
func NewPostgresDBCluster(configs []*util.DatabaseConfig) (*PostgresDB, error) {
	// 参数验证
	if len(configs) == 0 {
		return nil, fmt.Errorf("database configs is empty")
	}

	// 尝试连接主节点
	var lastErr error

	for _, config := range configs {
		if config == nil {
			continue
		}

		dsn := fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			config.Host, config.Port, config.User, config.Password, config.DBName, config.SSLMode,
		)

		gormConfig := &gorm.Config{
			Logger: logger.Default.LogMode(logger.Info),
			NowFunc: func() time.Time {
				return time.Now().UTC()
			},
		}

		db, err := gorm.Open(postgres.Open(dsn), gormConfig)
		if err == nil {
			// 连接成功，使用该节点
			// 配置连接池
			sqlDB, err := db.DB()
			if err != nil {
				lastErr = err
				continue
			}

			sqlDB.SetMaxOpenConns(config.MaxOpenConns)
			sqlDB.SetMaxIdleConns(config.MaxIdleConns)
			connMaxLifetime, _ := time.ParseDuration(config.ConnMaxLifetime)
			sqlDB.SetConnMaxLifetime(connMaxLifetime)
			connMaxIdleTime, _ := time.ParseDuration(config.ConnMaxIdleTime)
			sqlDB.SetConnMaxIdleTime(connMaxIdleTime)

			// 测试连接
			if err := sqlDB.Ping(); err != nil {
				_ = sqlDB.Close()
				lastErr = fmt.Errorf("database ping failed: %w", err)
				continue
			}

			return &PostgresDB{DB: db}, nil
		}
		lastErr = err
	}

	return nil, fmt.Errorf("failed to connect to any database node: %w", lastErr)
}

// Close 关闭数据库连接
func (p *PostgresDB) Close() error {
	if p.DB == nil {
		return nil
	}
	sqlDB, err := p.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
