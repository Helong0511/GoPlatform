import React, { useState, useEffect } from 'react';
import { Card, Row, Col, Statistic, Spin, Alert } from 'antd';
import {
  UserOutlined,
  FileTextOutlined,
  RiseOutlined,
  CheckCircleOutlined,
} from '@ant-design/icons';
import { getUsers, getAdTasks, getTodayReports } from '../api';

/**
 * 仪表盘页面组件
 * 显示系统关键统计数据
 */
function Dashboard() {
  const [stats, setStats] = useState({
    userCount: 0,
    adCount: 0,
    activeAdCount: 0,
    todayReportCount: 0,
  });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    fetchData();
  }, []);

  /**
   * 获取统计数据
   */
  const fetchData = async () => {
    setLoading(true);
    setError(null);
    try {
      // 并行请求所有数据
      const [usersData, adsData, reportsData] = await Promise.all([
        getUsers(1, 1),
        getAdTasks(1, 1),
        getTodayReports(),
      ]);

      // 计算活跃广告数量
      const activeAds = adsData.data?.filter(ad => ad.status === 'active').length || 0;

      // 更新统计数据
      setStats({
        userCount: usersData.meta?.total || 0,
        adCount: adsData.meta?.total || 0,
        activeAdCount: activeAds,
        todayReportCount: reportsData.data?.count || 0,
      });
    } catch (error) {
      console.error('获取统计数据失败:', error);
      setError('获取数据失败，请稍后重试');
      // 显示默认数据
      setStats({
        userCount: 0,
        adCount: 0,
        activeAdCount: 0,
        todayReportCount: 0,
      });
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="page-container">
      <div className="page-header">
        <h2>仪表盘</h2>
      </div>
      
      {error && (
        <Alert 
          message="错误" 
          description={error} 
          type="error" 
          showIcon 
          style={{ marginBottom: 24 }}
          action={
            <button 
              onClick={fetchData}
              style={{
                border: '1px solid #ff4d4f',
                backgroundColor: 'white',
                color: '#ff4d4f',
                padding: '4px 12px',
                borderRadius: '4px',
                cursor: 'pointer',
              }}
            >
              重试
            </button>
          }
        />
      )}

      <Spin spinning={loading} tip="加载中...">
        <Row gutter={[16, 16]}>
          <Col xs={24} sm={12} lg={6}>
            <Card hoverable className="statistic-card">
              <Statistic
                title="用户总数"
                value={stats.userCount}
                prefix={<UserOutlined />}
                valueStyle={{ color: '#1890ff' }}
              />
            </Card>
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <Card hoverable className="statistic-card">
              <Statistic
                title="广告任务总数"
                value={stats.adCount}
                prefix={<FileTextOutlined />}
                valueStyle={{ color: '#52c41a' }}
              />
            </Card>
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <Card hoverable className="statistic-card">
              <Statistic
                title="活跃广告"
                value={stats.activeAdCount}
                prefix={<CheckCircleOutlined />}
                valueStyle={{ color: '#faad14' }}
              />
            </Card>
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <Card hoverable className="statistic-card">
              <Statistic
                title="今日上报"
                value={stats.todayReportCount}
                prefix={<RiseOutlined />}
                valueStyle={{ color: '#722ed1' }}
              />
            </Card>
          </Col>
        </Row>
      </Spin>
    </div>
  );
}

export default Dashboard;