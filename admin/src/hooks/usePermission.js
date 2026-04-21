import { useState, useEffect, useCallback } from 'react';

/**
 * 权限控制自定义Hook
 * 用于检查用户是否拥有特定权限
 * @returns {Object} 包含权限检查方法和权限列表的对象
 */
export const usePermission = () => {
  const [permissions, setPermissions] = useState([]);
  const [userRoles, setUserRoles] = useState([]);

  // 从会话存储获取并更新用户权限信息
  const updateUserPermissions = () => {
    try {
      const user = JSON.parse(sessionStorage.getItem('user') || '{}');
      setPermissions(Array.isArray(user.permissions) ? user.permissions : []);
      setUserRoles(Array.isArray(user.roles) ? user.roles : []);
    } catch (error) {
      console.error('解析用户权限信息失败:', error);
      setPermissions([]);
      setUserRoles([]);
    }
  };

  useEffect(() => {
    // 初始化时更新权限信息
    updateUserPermissions();

    // 监听 localStorage 变化
    const handleStorageChange = (e) => {
      if (e.key === 'user' || e.key === null) {
        updateUserPermissions();
      }
    };

    // 添加事件监听器
    window.addEventListener('storage', handleStorageChange);

    // 添加定时器定期更新权限信息
    const updateInterval = setInterval(() => {
      updateUserPermissions();
    }, 5000); // 每 5 秒检查一次

    // 清理事件监听器和定时器
    return () => {
      window.removeEventListener('storage', handleStorageChange);
      clearInterval(updateInterval);
    };
  }, []);

  /**
   * 检查用户是否拥有指定权限
   * @param {string} permission - 权限名称
   * @returns {boolean} 是否拥有该权限
   */
  const hasPermission = useCallback((permission) => {
    return permissions.includes(permission);
  }, [permissions]);

  /**
   * 检查用户是否拥有指定角色
   * @param {string} role - 角色名称
   * @returns {boolean} 是否拥有该角色
   */
  const hasRole = useCallback((role) => {
    return userRoles.some(r => r.name === role);
  }, [userRoles]);

  /**
   * 检查用户是否拥有指定资源的操作权限
   * @param {string} resource - 资源名称
   * @param {string} action - 操作类型 (create, read, update, delete)
   * @returns {boolean} 是否拥有该操作权限
   */
  const can = useCallback((resource, action) => {
    const permission = `${resource}:${action}`;
    return hasPermission(permission);
  }, [hasPermission]);

  return {
    permissions,
    userRoles,
    hasPermission,
    hasRole,
    can,
  };
};

export default usePermission;