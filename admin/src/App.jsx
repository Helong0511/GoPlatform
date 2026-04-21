import React, { useState, useEffect, useCallback, lazy, Suspense } from 'react';
import { Routes, Route, Navigate, useNavigate } from 'react-router-dom';
import { Layout, Spin } from 'antd';
import Login from './pages/Login';
import AdminLayout from './components/AdminLayout.jsx';
import { setUserTimezone } from './utils/datetime';

// 懒加载组件
const Dashboard = lazy(() => import('./pages/Dashboard'));
const UserManagement = lazy(() => import('./pages/UserManagement'));
const RoleManagement = lazy(() => import('./pages/RoleManagement'));
const AdManagement = lazy(() => import('./pages/AdManagement'));
const DepartmentManagement = lazy(() => import('./pages/DepartmentManagement'));
const ClientManagement = lazy(() => import('./pages/ClientManagement'));
const AdReportManagement = lazy(() => import('./pages/AdReportManagement'));

function App() {
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const navigate = useNavigate();

  // 检查认证状态
  const checkAuthStatus = useCallback(() => {
    const token = sessionStorage.getItem('token');
    if (token) {
      setIsAuthenticated(true);
    } else {
      setIsAuthenticated(false);
      // 如果没有 token，跳转到登录页面
      navigate('/login');
    }
  }, [navigate]);

  useEffect(() => {
    // 初始化时检查认证状态
    checkAuthStatus();

    // 监听 localStorage 变化
    const handleStorageChange = (e) => {
      if (e.key === 'token' || e.key === null) {
        checkAuthStatus();
      }
    };

    // 添加事件监听器
    window.addEventListener('storage', handleStorageChange);

    // 添加定时器定期检查认证状态（解决当前窗口清除数据的问题）
    const checkAuthInterval = setInterval(() => {
      checkAuthStatus();
    }, 5000); // 每 5 秒检查一次

    // 清理事件监听器和定时器
    return () => {
      window.removeEventListener('storage', handleStorageChange);
      clearInterval(checkAuthInterval);
    };
  }, [navigate, checkAuthStatus]);

  const handleLogin = (token, user) => {
    try {
      sessionStorage.setItem('token', token);
      // 确保user是对象
      const userObj = typeof user === 'object' && user !== null ? user : {};
      sessionStorage.setItem('user', JSON.stringify(userObj));
      // 存储用户时区，用于时间转换
      setUserTimezone(userObj.timezone || 'Asia/Shanghai');
      setIsAuthenticated(true);
      navigate('/');
    } catch (error) {
      console.error('登录处理失败:', error);
      // 即使出错也要设置认证状态，避免无限循环
      setIsAuthenticated(false);
      navigate('/login');
    }
  };

  const handleLogout = () => {
    sessionStorage.removeItem('token');
    sessionStorage.removeItem('user');
    setIsAuthenticated(false);
    navigate('/login');
  };

  if (!isAuthenticated) {
    return (
      <Routes>
        <Route path="/login" element={<Login onLogin={handleLogin} />} />
        <Route path="*" element={<Navigate to="/login" replace />} />
      </Routes>
    );
  }

  return (
    <Routes>
      <Route path="/login" element={<Navigate to="/" replace />} />
      <Route path="/" element={<AdminLayout onLogout={handleLogout} />}>
        <Route index element={<Dashboard />} />
        <Route path="departments" element={<DepartmentManagement />} />
        <Route path="roles" element={<RoleManagement />} />
        <Route path="users" element={<UserManagement />} />
        <Route path="clients" element={<ClientManagement />} />
        <Route path="ads" element={<AdManagement />} />
        <Route path="ad-reports" element={<AdReportManagement />} />
      </Route>
    </Routes>
  );
}

export default App;