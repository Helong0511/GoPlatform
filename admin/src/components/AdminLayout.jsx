import React, { useState, useMemo, lazy, Suspense } from 'react';
import { useNavigate, useLocation, Outlet } from 'react-router-dom';
import { Layout, Menu, Avatar, Dropdown, theme, Tooltip, Spin } from 'antd';
import {
  DashboardOutlined,
  UserOutlined,
  TeamOutlined,
  FileTextOutlined,
  ApartmentOutlined,
  ShopOutlined,
  RiseOutlined,
  SafetyCertificateOutlined,
  SettingOutlined,
  LogoutOutlined,
  MenuUnfoldOutlined,
  MenuFoldOutlined,
} from '@ant-design/icons';
import usePermission from '../hooks/usePermission';

const { Header, Sider, Content } = Layout;

// 导航菜单项配置（带权限控制）
const menuItems = [
  {
    key: '/',
    icon: <DashboardOutlined />,
    label: '仪表盘',
    permission: 'report:read',
  },
  {
    key: '/departments',
    icon: <ApartmentOutlined />,
    label: '部门管理',
    permission: 'department:read',
  },
  {
    key: '/roles',
    icon: <TeamOutlined />,
    label: '角色管理',
    permission: 'role:read',
  },
  {
    key: '/users',
    icon: <UserOutlined />,
    label: '用户管理',
    permission: 'user:read',
  },
  {
    key: '/clients',
    icon: <ShopOutlined />,
    label: '客户管理',
    permission: 'client:read',
  },
  {
    key: '/ads',
    icon: <FileTextOutlined />,
    label: '广告管理',
    permission: 'ad:read',
  },
  {
    key: '/ad-reports',
    icon: <RiseOutlined />,
    label: '广告上报',
    permission: 'report:read',
  },
  {
    key: '/ip-access',
    icon: <SafetyCertificateOutlined />,
    label: 'IP 访问控制',
    permission: 'user:read',
  },
  {
    key: '/system-config',
    icon: <SettingOutlined />,
    label: '系统配置',
    permission: 'user:read',
  },
];

/**
 * 管理后台布局组件
 * @param {Object} props - 组件属性
 * @param {React.ReactNode} props.children - 子组件
 * @param {Function} props.onLogout - 退出登录回调函数
 */
function AdminLayout({ children, onLogout }) {
  // 侧边栏折叠状态
  const [collapsed, setCollapsed] = useState(false);
  const navigate = useNavigate();
  const location = useLocation();
  const {
    token: { colorBgContainer, borderRadiusLG },
  } = theme.useToken();

  // 从会话存储获取用户信息
  const user = useMemo(() => {
    try {
      return JSON.parse(sessionStorage.getItem('user') || '{}');
    } catch (error) {
      console.error('解析用户信息失败:', error);
      return {};
    }
  }, []);
  
  // 使用权限Hook
  const { hasPermission } = usePermission();
  
  // 根据权限过滤菜单项
  const filteredMenuItems = useMemo(() => {
    return menuItems.filter(item => {
      // 如果没有配置权限，则默认显示
      if (!item.permission) return true;
      // 检查用户是否拥有该权限
      return hasPermission(item.permission);
    });
  }, [hasPermission]);

  // 用户菜单配置
  const userMenuItems = [
    {
      key: 'logout',
      icon: <LogoutOutlined />,
      label: '退出登录',
      onClick: onLogout,
    },
  ];

  /**
   * 处理菜单点击事件
   * @param {Object} e - 点击事件对象
   */
  const handleMenuClick = ({ key }) => {
    navigate(key);
  };

  return (
    <Layout style={{ minHeight: '100vh' }}>
      {/* 侧边栏 */}
      <Sider 
        trigger={window.innerWidth > 768 ? null : 'responsive'}
        collapsible 
        collapsed={collapsed}
        width={200}
        collapsedWidth={80}
        breakpoint="md"
        style={{
          background: '#001529',
          transition: 'all 0.3s',
        }}
      >
        {/* Logo区域 */}
        <div 
          className="logo" 
          style={{ 
            justifyContent: 'center',
            height: 64,
            display: 'flex',
            alignItems: 'center',
            color: 'white',
            fontSize: collapsed ? '18px' : '20px',
            fontWeight: 'bold',
            transition: 'all 0.3s',
          }}
        >
          {collapsed ? 'AD' : '广告平台'}
        </div>
        {/* 导航菜单 */}
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[location.pathname]}
          items={filteredMenuItems}
          onClick={handleMenuClick}
          style={{ borderRight: 0 }}
        />
      </Sider>
      <Layout>
        {/* 头部区域 */}
        <Header 
          style={{ 
            padding: '0 16px', 
            background: colorBgContainer, 
            display: 'flex', 
            justifyContent: 'space-between', 
            alignItems: 'center',
            boxShadow: '0 2px 8px rgba(0, 0, 0, 0.09)',
            zIndex: 10,
          }}
        >
          {/* 侧边栏折叠按钮 */}
          <Tooltip title={collapsed ? '展开菜单' : '收起菜单'}>
            <div 
              onClick={() => setCollapsed(!collapsed)} 
              style={{ 
                cursor: 'pointer', 
                fontSize: '18px',
                padding: '8px',
                borderRadius: '4px',
                transition: 'all 0.3s',
              }}
              onMouseEnter={(e) => e.target.style.backgroundColor = 'rgba(0, 0, 0, 0.05)'}
              onMouseLeave={(e) => e.target.style.backgroundColor = 'transparent'}
            >
              {collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
            </div>
          </Tooltip>
          {/* 用户信息和退出登录 */}
          <Dropdown menu={{ items: userMenuItems }} placement="bottomRight">
            <div 
              style={{ 
                cursor: 'pointer', 
                display: 'flex', 
                alignItems: 'center', 
                gap: '8px',
                padding: '8px 12px',
                borderRadius: '4px',
                transition: 'all 0.3s',
              }}
              onMouseEnter={(e) => e.currentTarget.style.backgroundColor = 'rgba(0, 0, 0, 0.05)'}
              onMouseLeave={(e) => e.currentTarget.style.backgroundColor = 'transparent'}
            >
              <Avatar icon={<UserOutlined />} />
              <span>{user.username || '管理员'}</span>
            </div>
          </Dropdown>
        </Header>
        {/* 内容区域 */}
        <Content 
          style={{ 
            margin: '16px', 
            padding: 24, 
            background: colorBgContainer, 
            borderRadius: borderRadiusLG, 
            minHeight: 280,
            boxShadow: '0 2px 8px rgba(0, 0, 0, 0.06)',
          }}
        >
          <Suspense fallback={<div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: 200 }}><Spin size="large" /></div>}>
            <Outlet />
          </Suspense>
        </Content>
      </Layout>
    </Layout>
  );
}

export default AdminLayout;