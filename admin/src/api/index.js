import axios from 'axios';
import { config } from '../config';
import { getUserTimezone } from '../utils/datetime';

/**
 * API客户端实例
 * 配置基础URL、超时时间和默认请求头
 */
const apiClient = axios.create({
  baseURL: config.api.baseUrl,
  timeout: config.api.timeout,
  headers: {
    'Content-Type': 'application/json',
  },
});

/**
 * 请求拦截器
 * 添加认证token和时区信息到请求头
 */
apiClient.interceptors.request.use(
  (config) => {
    const token = sessionStorage.getItem('token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    // 添加用户时区信息到请求头
    const timezone = getUserTimezone();
    config.headers['X-Timezone'] = timezone;
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

/**
 * 响应拦截器
 * 处理错误响应，特别是401未授权错误
 */
apiClient.interceptors.response.use(
  (response) => {
    // 后端返回的是统一响应格式，包含code、message、data等字段
    // 成功响应时，返回data字段
    if (response.data.code === 200) {
      return response.data;
    }
    // 非成功响应，作为错误处理
    return Promise.reject(response.data.message || '请求失败');
  },
  (error) => {
    if (error.response) {
      const { status, data } = error.response;
      if (status === 401) {
        // 清除会话存储的token和用户信息
        sessionStorage.removeItem('token');
        sessionStorage.removeItem('user');
        // 只有在非登录页面才跳转，避免登录失败时的跳转导致错误提示无法显示
        if (window.location.pathname !== '/login') {
          window.location.href = '/login';
        }
      }
      // 后端返回的错误响应也有统一格式
      return Promise.reject(data.message || '请求失败');
    }
    // 处理网络错误，确保Promise正确拒绝
    return Promise.reject(error.message || '网络错误');
  }
);

// 登录
export const login = (username, password) => {
  return apiClient.post('/login', { username, password });
};

// 获取用户列表
export const getUsers = (page = 1, pageSize = 10, filters = {}) => {
  const params = { page, page_size: pageSize, ...filters };
  return apiClient.get('/admin/users', { params });
};

// 获取单个用户
export const getUser = (id) => {
  return apiClient.get(`/admin/users/${id}`);
};

// 创建用户
export const createUser = (data) => {
  return apiClient.post('/admin/users', data);
};

// 更新用户
export const updateUser = (id, data) => {
  return apiClient.put(`/admin/users/${id}`, data);
};

// 删除用户
export const deleteUser = (id) => {
  return apiClient.delete(`/admin/users/${id}`);
};

// 启用用户
export const enableUser = (id) => {
  return apiClient.put(`/admin/users/${id}/enable`);
};

// 禁用用户
export const disableUser = (id) => {
  return apiClient.put(`/admin/users/${id}/disable`);
};

// 获取用户状态变更日志
export const getUserStatusLogs = (id, page = 1, pageSize = 10) => {
  return apiClient.get(`/admin/users/${id}/status-logs?page=${page}&page_size=${pageSize}`);
};

// 获取角色列表
export const getRoles = () => {
  return apiClient.get('/admin/roles');
};

// 获取单个角色
export const getRole = (id) => {
  return apiClient.get(`/admin/roles/${id}`);
};

// 创建角色
export const createRole = (data) => {
  return apiClient.post('/admin/roles', data);
};

// 更新角色
export const updateRole = (id, data) => {
  return apiClient.put(`/admin/roles/${id}`, data);
};

// 删除角色
export const deleteRole = (id) => {
  return apiClient.delete(`/admin/roles/${id}`);
};

// 分配权限
export const assignPermissions = (id, permissionIds) => {
  return apiClient.post(`/admin/roles/${id}/permissions`, { permission_ids: permissionIds });
};

// 分配角色
export const assignRoles = (userId, roleIds) => {
  return apiClient.post('/admin/roles/assign', { user_id: userId, role_ids: roleIds });
};

// 获取权限列表
export const getPermissions = () => {
  return apiClient.get('/admin/permissions');
};

// 获取广告任务列表
export const getAdTasks = (page = 1, pageSize = 10, filters = {}) => {
  const params = { page, page_size: pageSize, ...filters };
  return apiClient.get('/admin/ads', { params });
};

// 获取单个广告任务
export const getAdTask = (id) => {
  return apiClient.get(`/admin/ads/${id}`);
};

// 创建广告任务
export const createAdTask = (data) => {
  return apiClient.post('/admin/ads', data);
};

// 更新广告任务
export const updateAdTask = (id, data) => {
  return apiClient.put(`/admin/ads/${id}`, data);
};

// 删除广告任务
export const deleteAdTask = (id) => {
  return apiClient.delete(`/admin/ads/${id}`);
};

// 获取部门列表
export const getDepartments = (active = false) => {
  return apiClient.get(`/admin/departments?active=${active}`);
};

// 获取单个部门
export const getDepartment = (id) => {
  return apiClient.get(`/admin/departments/${id}`);
};

// 创建部门
export const createDepartment = (data) => {
  return apiClient.post('/admin/departments', data);
};

// 更新部门
export const updateDepartment = (id, data) => {
  return apiClient.put(`/admin/departments/${id}`, data);
};

// 删除部门
export const deleteDepartment = (id) => {
  return apiClient.delete(`/admin/departments/${id}`);
};

// 获取客户列表
export const getClients = (page = 1, pageSize = 10, filters = {}) => {
  const params = { page, page_size: pageSize, ...filters };
  return apiClient.get('/admin/clients', { params });
};

// 获取单个客户
export const getClient = (id) => {
  return apiClient.get(`/admin/clients/${id}`);
};

// 创建客户
export const createClient = (data) => {
  return apiClient.post('/admin/clients', data);
};

// 更新客户
export const updateClient = (id, data) => {
  return apiClient.put(`/admin/clients/${id}`, data);
};

// 删除客户
export const deleteClient = (id) => {
  return apiClient.delete(`/admin/clients/${id}`);
};

// 获取今日上报数据
export const getTodayReports = () => {
  return apiClient.get('/admin/reports/today');
};

// 获取广告上报列表
export const getAdReports = (page = 1, pageSize = 10, filters = {}) => {
  const params = { page, page_size: pageSize, ...filters };
  return apiClient.get('/admin/reports', { params });
};

// 批量添加上报数据
export const batchCreateAdReports = (reports) => {
  return apiClient.post('/admin/reports/batch', { reports });
};

// 获取上报统计数据
export const getReportStats = () => {
  return apiClient.get('/admin/reports/stats');
};

// 按时间范围查询上报数据
export const getAdReportsByDateRange = (startDate, endDate, filters = {}) => {
  const params = { start_date: startDate, end_date: endDate, ...filters };
  return apiClient.get('/admin/reports/range', { params });
};

// 获取活跃广告任务（使用物化视图）
export const getActiveAdTasks = (page = 1, pageSize = 10, filters = {}) => {
  const params = { page, page_size: pageSize, ...filters };
  return apiClient.get('/admin/ads/active', { params });
};

// 批量更新广告任务
export const batchUpdateAdTasks = (ids, data) => {
  return apiClient.put('/admin/ads/batch', { ids, data });
};

// IP 访问控制
export const getIPAccessStatus = () => {
  return apiClient.get('/admin/system/ip-access/status');
};

export const getIPBlacklist = () => {
  return apiClient.get('/admin/system/ip-blacklist');
};

export const addIPBlacklist = (data) => {
  return apiClient.post('/admin/system/ip-blacklist', data);
};

export const removeIPBlacklist = (ip) => {
  return apiClient.delete(`/admin/system/ip-blacklist/${encodeURIComponent(ip)}`);
};

export const getIPWhitelist = () => {
  return apiClient.get('/admin/system/ip-whitelist');
};

export const addIPWhitelist = (data) => {
  return apiClient.post('/admin/system/ip-whitelist', data);
};

export const removeIPWhitelist = (ip) => {
  return apiClient.delete(`/admin/system/ip-whitelist/${encodeURIComponent(ip)}`);
};

export const getSystemConfig = () => apiClient.get('/admin/system/config');
export const saveSystemConfig = (payload) => apiClient.put('/admin/system/config', payload);
export const applyHotSystemConfig = () => apiClient.post('/admin/system/config/apply-hot');

export default apiClient;
