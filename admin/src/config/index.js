// 系统配置管理

/**
 * 系统配置
 */
export const config = {
  // 密码要求设置
  password: {
    // 是否启用严格密码要求（优先从环境变量读取，默认true）
    strictMode: process.env.REACT_APP_PASSWORD_STRICT_MODE !== 'false',
    // 密码最小长度
    minLength: 6,
    // 密码最大长度
    maxLength: 20,
    // 严格模式下的要求
    strictRequirements: {
      // 是否要求大写字母
      requireUppercase: true,
      // 是否要求小写字母
      requireLowercase: true,
      // 是否要求数字
      requireNumber: true,
      // 是否要求特殊字符
      requireSpecialChar: false
    }
  },
  // API配置
  api: {
    // API基础URL
    baseUrl: process.env.REACT_APP_API_URL || '/api',
    // 请求超时时间（毫秒）
    timeout: 30000
  },
  // 系统设置
  system: {
    // 是否启用调试模式
    debug: process.env.NODE_ENV === 'development'
  }
};

/**
 * 获取密码验证规则
 * @returns {Array} 密码验证规则数组
 */
export const getPasswordRules = () => {
  const rules = [
    { required: true, message: '请输入密码' },
    { min: config.password.minLength, message: `密码至少${config.password.minLength}个字符` },
    { max: config.password.maxLength, message: `密码最多${config.password.maxLength}个字符` }
  ];

  // 如果启用严格模式，添加额外的验证规则
  if (config.password.strictMode) {
    rules.push({
      validator: (_, value) => {
        if (!value) {
          return Promise.resolve();
        }
        
        if (config.password.strictRequirements.requireUppercase && !/[A-Z]/.test(value)) {
          return Promise.reject(new Error('密码必须包含至少一个大写字母'));
        }
        
        if (config.password.strictRequirements.requireLowercase && !/[a-z]/.test(value)) {
          return Promise.reject(new Error('密码必须包含至少一个小写字母'));
        }
        
        if (config.password.strictRequirements.requireNumber && !/[0-9]/.test(value)) {
          return Promise.reject(new Error('密码必须包含至少一个数字'));
        }
        
        if (config.password.strictRequirements.requireSpecialChar && !/[^a-zA-Z0-9]/.test(value)) {
          return Promise.reject(new Error('密码必须包含至少一个特殊字符'));
        }
        
        return Promise.resolve();
      }
    });
  }

  return rules;
};

export default config;