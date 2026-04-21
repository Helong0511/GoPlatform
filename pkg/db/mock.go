package db

import "context"

// MockDB 模拟数据库用于测试
type MockDB struct {
	IsConnected bool
	Error       error
}

// NewMockDB 创建模拟数据库
func NewMockDB() *MockDB {
	return &MockDB{
		IsConnected: true,
		Error:       nil,
	}
}

// Ping 测试数据库连接
func (m *MockDB) Ping(ctx context.Context) error {
	if !m.IsConnected {
		return m.Error
	}
	return nil
}

// Close 关闭数据库连接
func (m *MockDB) Close() error {
	m.IsConnected = false
	return nil
}

// Query 执行查询(模拟)
func (m *MockDB) Query(sql string, args ...interface{}) error {
	if !m.IsConnected {
		return m.Error
	}
	return nil
}

// Exec 执行命令(模拟)
func (m *MockDB) Exec(sql string, args ...interface{}) error {
	if !m.IsConnected {
		return m.Error
	}
	return nil
}
