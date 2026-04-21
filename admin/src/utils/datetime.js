import dayjs from 'dayjs';
import utc from 'dayjs/plugin/utc';
import timezone from 'dayjs/plugin/timezone';

// 扩展 dayjs 插件
dayjs.extend(utc);
dayjs.extend(timezone);

/**
 * 获取用户时区
 * 优先级：sessionStorage 中的用户时区 > 默认时区 (Asia/Shanghai)
 * @returns {string} 用户时区，如 'Asia/Shanghai'
 */
export const getUserTimezone = () => {
  // 尝试从 sessionStorage 获取用户信息
  const userStr = sessionStorage.getItem('user');
  if (userStr) {
    try {
      const user = JSON.parse(userStr);
      if (user.timezone) {
        return user.timezone;
      }
    } catch (e) {
      // 解析失败，使用默认值
    }
  }
  
  // 尝试从 sessionStorage 获取单独存储的时区
  const storedTimezone = sessionStorage.getItem('userTimezone');
  if (storedTimezone) {
    return storedTimezone;
  }
  
  // 默认使用上海时区
  return 'Asia/Shanghai';
};

/**
 * 设置用户时区到 sessionStorage
 * @param {string} tz - 时区字符串，如 'Asia/Shanghai'
 */
export const setUserTimezone = (tz) => {
  if (tz) {
    sessionStorage.setItem('userTimezone', tz);
  }
};

/**
 * 将用户本地时间转换为 UTC ISO 字符串（用于 API 请求）
 * 用户在前端选择的时间会被当作用户时区的时间，然后转换为 UTC 发送给后端
 * 
 * @param {Date|string|dayjs.Dayjs|number} date - 用户选择的时间
 * @returns {string} UTC ISO 字符串，如 '2024-01-01T00:00:00.000Z'
 * 
 * @example
 * // 用户在上海时区选择了 2024-01-01 08:00:00
 * // 转换后应为 UTC 时间 2024-01-01 00:00:00
 * toUTC('2024-01-01 08:00:00') // '2024-01-01T00:00:00.000Z'
 */
export const toUTC = (date) => {
  if (!date) return null;
  const tz = getUserTimezone();
  // 将输入时间当作用户时区的时间，然后转换为 UTC
  return dayjs.tz(date, tz).utc().toISOString();
};

/**
 * 将 UTC 时间转换为用户时区显示
 * 后端返回的 UTC 时间会被转换为用户时区的时间显示
 * 
 * @param {string|Date|number} utcTime - UTC 时间
 * @param {string} format - 格式化模板，默认 'YYYY-MM-DD HH:mm:ss'
 * @returns {string} 格式化后的用户时区时间
 * 
 * @example
 * // 用户在上海时区 (UTC+8)
 * // 后端返回 UTC 时间 '2024-01-01T00:00:00.000Z'
 * // 转换后应为上海时间 2024-01-01 08:00:00
 * toUserTimezone('2024-01-01T00:00:00.000Z') // '2024-01-01 08:00:00'
 */
export const toUserTimezone = (utcTime, format = 'YYYY-MM-DD HH:mm:ss') => {
  if (!utcTime) return '-';
  const tz = getUserTimezone();
  try {
    return dayjs.utc(utcTime).tz(tz).format(format);
  } catch (e) {
    return '-';
  }
};

/**
 * 获取用户时区的当前时间
 * @returns {dayjs.Dayjs} 当前时间（用户时区）
 */
export const nowInUserTimezone = () => {
  const tz = getUserTimezone();
  return dayjs().tz(tz);
};

/**
 * 获取用户时区今天的开始时间（00:00:00）
 * @returns {dayjs.Dayjs} 今天开始时间（用户时区）
 */
export const startOfTodayInUserTimezone = () => {
  const tz = getUserTimezone();
  return dayjs().tz(tz).startOf('day');
};

/**
 * 获取用户时区今天的结束时间（23:59:59）
 * @returns {dayjs.Dayjs} 今天结束时间（用户时区）
 */
export const endOfTodayInUserTimezone = () => {
  const tz = getUserTimezone();
  return dayjs().tz(tz).endOf('day');
};

/**
 * 将 dayjs 对象转换为用户时区
 * @param {dayjs.Dayjs|Date|string} date - 时间对象
 * @returns {dayjs.Dayjs} 用户时区的 dayjs 对象
 */
export const toUserTimezoneDayjs = (date) => {
  if (!date) return null;
  const tz = getUserTimezone();
  return dayjs(date).tz(tz);
};

/**
 * 获取可用的时区列表
 * @returns {Array<{value: string, label: string, offset: string}>} 时区列表
 */
export const getAvailableTimezones = () => {
  return [
    { value: 'Asia/Shanghai', label: '中国标准时间 (北京)', offset: 'UTC+8' },
    { value: 'Asia/Hong_Kong', label: '香港时间', offset: 'UTC+8' },
    { value: 'Asia/Taipei', label: '台北时间', offset: 'UTC+8' },
    { value: 'Asia/Tokyo', label: '日本标准时间', offset: 'UTC+9' },
    { value: 'Asia/Seoul', label: '韩国标准时间', offset: 'UTC+9' },
    { value: 'Asia/Singapore', label: '新加坡时间', offset: 'UTC+8' },
    { value: 'Asia/Dubai', label: '阿联酋时间', offset: 'UTC+4' },
    { value: 'Europe/London', label: '伦敦时间', offset: 'UTC+0/+1' },
    { value: 'Europe/Paris', label: '巴黎时间', offset: 'UTC+1/+2' },
    { value: 'Europe/Berlin', label: '柏林时间', offset: 'UTC+1/+2' },
    { value: 'Europe/Moscow', label: '莫斯科时间', offset: 'UTC+3' },
    { value: 'America/New_York', label: '纽约时间', offset: 'UTC-5/-4' },
    { value: 'America/Los_Angeles', label: '洛杉矶时间', offset: 'UTC-8/-7' },
    { value: 'America/Chicago', label: '芝加哥时间', offset: 'UTC-6/-5' },
    { value: 'Australia/Sydney', label: '悉尼时间', offset: 'UTC+10/+11' },
    { value: 'Pacific/Auckland', label: '奥克兰时间', offset: 'UTC+12/+13' },
    { value: 'UTC', label: '协调世界时 (UTC)', offset: 'UTC+0' },
  ];
};

// 导出 dayjs 供其他模块使用
export { dayjs };
