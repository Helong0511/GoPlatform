import React, { useState, useEffect, useCallback } from 'react';
import {
  Tabs,
  Form,
  Input,
  InputNumber,
  Switch,
  Button,
  message,
  Space,
  Alert,
  Typography,
  Select,
  Divider,
  Tag,
} from 'antd';
import {
  ReloadOutlined,
  SaveOutlined,
  ThunderboltOutlined,
  SettingOutlined,
} from '@ant-design/icons';
import {
  getSystemConfig,
  saveSystemConfig,
  applyHotSystemConfig,
} from '../api';
import usePermission from '../hooks/usePermission';

const { Title, Text, Paragraph } = Typography;
const { TextArea } = Input;

const SECRET_MASK = '***';

function splitLines(text) {
  if (!text) return [];
  if (Array.isArray(text)) return text;
  return String(text)
    .split('\n')
    .map((s) => s.trim())
    .filter(Boolean);
}

function registerTripletToForm(val) {
  if (val === true) return 'true';
  if (val === false) return 'false';
  return 'default';
}

function SystemConfigManagement() {
  const { can } = usePermission();
  const canUpdate = can('user', 'update');
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [meta, setMeta] = useState(null);
  const [secretsForm] = Form.useForm();

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await getSystemConfig();
      const data = res.data || {};
      setMeta(data);
      const file = data.file || {};
      form.setFieldsValue({
        environment: file.environment,
        server: {
          ...(file.server || {}),
          trusted_proxies: (file.server?.trusted_proxies || []).join('\n'),
        },
        auth: {
          allow_public_register: registerTripletToForm(file.auth?.allow_public_register),
          min_password_length: file.auth?.min_password_length,
        },
        email: { ...file.email, enabled: !!file.email?.enabled },
        database: file.database || {},
        redis: file.redis || {},
        pulsar: file.pulsar || {},
        jwt: {
          ...file.jwt,
          secret: file.jwt?.secret === SECRET_MASK ? SECRET_MASK : '',
        },
        security: {
          ...file.security,
          aes_secret: file.security?.aes_secret === SECRET_MASK ? SECRET_MASK : '',
          ip_blacklist: {
            ...(file.security?.ip_blacklist || {}),
            static_ips: (file.security?.ip_blacklist?.static_ips || []).join('\n'),
            static_whitelist_ips: (file.security?.ip_blacklist?.static_whitelist_ips || []).join('\n'),
          },
        },
        metrics: {
          ...file.metrics,
          bearer_token: file.metrics?.bearer_token === SECRET_MASK ? SECRET_MASK : '',
        },
        logging: file.logging || {},
        cors: {
          allow_origins: (file.cors?.allow_origins || []).join('\n'),
        },
        rate_limit: file.rate_limit || {},
      });
      secretsForm.resetFields();
    } catch (e) {
      message.error(e || '加载配置失败');
    } finally {
      setLoading(false);
    }
  }, [form, secretsForm]);

  useEffect(() => {
    load();
  }, [load]);

  const buildPayload = () => {
    const v = form.getFieldsValue(true);
    const ar = v.auth?.allow_public_register;
    const allowReg = ar === 'true' ? true : ar === 'false' ? false : null;
    const origins = (v.cors?.allow_origins || '')
      .split('\n')
      .map((s) => s.trim())
      .filter(Boolean);

    const config = {
      environment: v.environment,
      server: {
        ...v.server,
        trusted_proxies: splitLines(v.server?.trusted_proxies),
      },
      auth: {
        allow_public_register: allowReg,
        min_password_length: v.auth?.min_password_length,
      },
      email: v.email,
      database: {
        ...v.database,
        password: v.database?.password === SECRET_MASK ? SECRET_MASK : v.database?.password || '',
      },
      redis: {
        ...v.redis,
        password: v.redis?.password === SECRET_MASK ? SECRET_MASK : v.redis?.password || '',
      },
      pulsar: v.pulsar,
      jwt: {
        ...v.jwt,
        secret: v.jwt?.secret === SECRET_MASK ? SECRET_MASK : v.jwt?.secret || '',
      },
      security: {
        ...v.security,
        aes_secret:
          v.security?.aes_secret === SECRET_MASK ? SECRET_MASK : v.security?.aes_secret || '',
        ip_blacklist: {
          ...v.security?.ip_blacklist,
          static_ips: splitLines(v.security?.ip_blacklist?.static_ips),
          static_whitelist_ips: splitLines(v.security?.ip_blacklist?.static_whitelist_ips),
        },
      },
      metrics: {
        ...v.metrics,
        bearer_token:
          v.metrics?.bearer_token === SECRET_MASK ? SECRET_MASK : v.metrics?.bearer_token || '',
      },
      logging: v.logging,
      cors: { allow_origins: origins.length ? origins : ['*'] },
      rate_limit: v.rate_limit,
    };

    const sec = secretsForm.getFieldsValue();
    const secrets = {};
    const pick = (key, field) => {
      const val = sec[field];
      if (val && val.trim() && val !== SECRET_MASK) {
        secrets[key] = val.trim();
      }
    };
    pick('jwt_secret', 'jwt_secret');
    pick('aes_secret', 'aes_secret');
    pick('database_password', 'database_password');
    pick('redis_password', 'redis_password');
    pick('email_password', 'email_password');
    pick('metrics_bearer_token', 'metrics_bearer_token');

    return { config, secrets: Object.keys(secrets).length ? secrets : undefined };
  };

  const onSave = async () => {
    try {
      await form.validateFields();
      setSaving(true);
      const payload = buildPayload();
      const res = await saveSystemConfig(payload);
      message.success('配置已保存');
      if (res.data?.restart_needed) {
        message.warning(res.data.restart_note || '部分配置需重启服务');
      }
      if (res.data?.hot_reload) {
        const hr = Object.entries(res.data.hot_reload)
          .map(([k, v]) => `${k}: ${v}`)
          .join(', ');
        message.info(`热更新: ${hr}`);
      }
      load();
    } catch (e) {
      if (e?.errorFields) return;
      message.error(e || '保存失败');
    } finally {
      setSaving(false);
    }
  };

  const onHotReload = async () => {
    try {
      const res = await applyHotSystemConfig();
      const hr = res.data?.hot_reload || {};
      message.success(`热更新完成: ${JSON.stringify(hr)}`);
    } catch (e) {
      message.error(e || '热更新失败');
    }
  };

  const tabItems = [
    {
      key: 'general',
      label: '运行环境',
      children: (
        <>
          <Form.Item name="environment" label="environment" rules={[{ required: true }]}>
            <Select
              options={[
                { value: 'development', label: 'development' },
                { value: 'test', label: 'test' },
                { value: 'production', label: 'production' },
                { value: 'staging', label: 'staging' },
              ]}
            />
          </Form.Item>
          <Title level={5}>HTTP 服务 (server)</Title>
          <Form.Item name={['server', 'host']} label="host">
            <Input placeholder="127.0.0.1" />
          </Form.Item>
          <Form.Item name={['server', 'port']} label="port">
            <InputNumber min={1} max={65535} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name={['server', 'timeout']} label="timeout">
            <Input placeholder="30s" />
          </Form.Item>
          <Form.Item name={['server', 'max_request_body_mb']} label="max_request_body_mb">
            <InputNumber min={1} max={512} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name={['server', 'shutdown_timeout']} label="shutdown_timeout">
            <Input placeholder="20s" />
          </Form.Item>
          <Form.Item name={['server', 'trusted_proxies']} label="trusted_proxies（每行一个）">
            <TextArea rows={2} placeholder="127.0.0.1" />
          </Form.Item>
        </>
      ),
    },
    {
      key: 'auth',
      label: '认证',
      children: (
        <>
          <Form.Item name={['auth', 'allow_public_register']} label="公开注册">
            <Select
              options={[
                { value: 'default', label: '按环境默认' },
                { value: 'true', label: '强制开启' },
                { value: 'false', label: '强制关闭' },
              ]}
            />
          </Form.Item>
          <Form.Item name={['auth', 'min_password_length']} label="最短密码长度">
            <InputNumber min={6} max={128} style={{ width: '100%' }} />
          </Form.Item>
        </>
      ),
    },
    {
      key: 'db',
      label: '数据库',
      children: (
        <>
          <Form.Item name={['database', 'driver']} label="driver">
            <Select options={[{ value: 'postgres' }, { value: 'mysql' }]} />
          </Form.Item>
          <Form.Item name={['database', 'host']} label="host">
            <Input />
          </Form.Item>
          <Form.Item name={['database', 'port']} label="port">
            <InputNumber style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name={['database', 'user']} label="user">
            <Input />
          </Form.Item>
          <Form.Item name={['database', 'dbname']} label="dbname">
            <Input />
          </Form.Item>
          <Form.Item name={['database', 'sslmode']} label="sslmode">
            <Input />
          </Form.Item>
          <Form.Item name={['database', 'max_open_conns']} label="max_open_conns">
            <InputNumber min={1} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name={['database', 'max_idle_conns']} label="max_idle_conns">
            <InputNumber min={0} style={{ width: '100%' }} />
          </Form.Item>
        </>
      ),
    },
    {
      key: 'redis',
      label: 'Redis',
      children: (
        <>
          <Form.Item name={['redis', 'host']} label="host">
            <Input />
          </Form.Item>
          <Form.Item name={['redis', 'port']} label="port">
            <InputNumber style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name={['redis', 'db']} label="db">
            <InputNumber min={0} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name={['redis', 'pool_size']} label="pool_size">
            <InputNumber min={1} style={{ width: '100%' }} />
          </Form.Item>
        </>
      ),
    },
    {
      key: 'pulsar',
      label: 'Pulsar',
      children: (
        <>
          <Form.Item name={['pulsar', 'url']} label="url（空=不启用）">
            <Input placeholder='留空跳过；如 pulsar://localhost:6650' />
          </Form.Item>
          <Form.Item name={['pulsar', 'topic']} label="topic">
            <Input />
          </Form.Item>
          <Form.Item name={['pulsar', 'subscription']} label="subscription">
            <Input />
          </Form.Item>
        </>
      ),
    },
    {
      key: 'jwt',
      label: 'JWT',
      children: (
        <>
          <Form.Item name={['jwt', 'expires_in']} label="expires_in">
            <Input placeholder="1h" />
          </Form.Item>
          <Form.Item name={['jwt', 'issuer']} label="issuer">
            <Input />
          </Form.Item>
          <Form.Item name={['jwt', 'audience']} label="audience">
            <Input />
          </Form.Item>
        </>
      ),
    },
    {
      key: 'security',
      label: '安全 / IP',
      children: (
        <>
          <Form.Item name={['security', 'rust_lib_path']} label="rust_lib_path">
            <Input />
          </Form.Item>
          <Form.Item name={['security', 'hsts_max_age_seconds']} label="hsts_max_age_seconds">
            <InputNumber min={0} style={{ width: '100%' }} />
          </Form.Item>
          <Divider>IP 黑名单（保存后可热更新）</Divider>
          <Form.Item
            name={['security', 'ip_blacklist', 'enabled']}
            label="启用自动黑名单"
            valuePropName="checked"
          >
            <Switch />
          </Form.Item>
          <Form.Item name={['security', 'ip_blacklist', 'ban_duration']} label="ban_duration">
            <Input placeholder="24h" />
          </Form.Item>
          <Form.Item name={['security', 'ip_blacklist', 'strike_window']} label="strike_window">
            <Input placeholder="10m" />
          </Form.Item>
          <Form.Item name={['security', 'ip_blacklist', 'strikes_before_ban']} label="strikes_before_ban">
            <InputNumber min={1} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item
            name={['security', 'ip_blacklist', 'block_crawler_user_agent']}
            label="拦截爬虫 UA"
            valuePropName="checked"
          >
            <Switch />
          </Form.Item>
          <Form.Item name={['security', 'ip_blacklist', 'crawler_strike_weight']} label="crawler_strike_weight">
            <InputNumber min={1} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name={['security', 'ip_blacklist', 'static_ips']} label="static_ips（每行一个）">
            <TextArea rows={2} />
          </Form.Item>
          <Form.Item name={['security', 'ip_blacklist', 'static_whitelist_ips']} label="static_whitelist_ips">
            <TextArea rows={2} />
          </Form.Item>
        </>
      ),
    },
    {
      key: 'metrics',
      label: '指标',
      children: (
        <>
          <Form.Item name={['metrics', 'host']} label="host">
            <Input />
          </Form.Item>
          <Form.Item name={['metrics', 'port']} label="port">
            <InputNumber style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name={['metrics', 'path']} label="path">
            <Input />
          </Form.Item>
        </>
      ),
    },
    {
      key: 'logging',
      label: '日志',
      children: (
        <>
          <Form.Item name={['logging', 'level']} label="level">
            <Select options={[{ value: 'debug' }, { value: 'info' }, { value: 'warn' }, { value: 'error' }]} />
          </Form.Item>
          <Form.Item name={['logging', 'format']} label="format">
            <Select options={[{ value: 'json' }, { value: 'console' }]} />
          </Form.Item>
          <Form.Item name={['logging', 'slow_request_ms']} label="slow_request_ms">
            <InputNumber min={0} style={{ width: '100%' }} />
          </Form.Item>
        </>
      ),
    },
    {
      key: 'cors',
      label: 'CORS',
      children: (
        <Form.Item name={['cors', 'allow_origins']} label="allow_origins（每行一个）">
          <TextArea rows={4} placeholder="https://admin.example.com" />
        </Form.Item>
      ),
    },
    {
      key: 'ratelimit',
      label: '限流',
      children: (
        <>
          <Form.Item name={['rate_limit', 'requests_per_second']} label="requests_per_second">
            <InputNumber min={0.1} step={1} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name={['rate_limit', 'burst']} label="burst">
            <InputNumber min={1} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name={['rate_limit', 'login_requests_per_second']} label="login_requests_per_second">
            <InputNumber min={0} step={0.1} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name={['rate_limit', 'login_burst']} label="login_burst">
            <InputNumber min={0} style={{ width: '100%' }} />
          </Form.Item>
        </>
      ),
    },
    {
      key: 'email',
      label: '邮件',
      children: (
        <>
          <Form.Item name={['email', 'enabled']} label="启用邮件" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item name={['email', 'host']} label="host">
            <Input />
          </Form.Item>
          <Form.Item name={['email', 'port']} label="port">
            <InputNumber style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name={['email', 'username']} label="username">
            <Input />
          </Form.Item>
          <Form.Item name={['email', 'from']} label="from">
            <Input />
          </Form.Item>
        </>
      ),
    },
  ];

  return (
    <div>
      <Title level={4} style={{ marginTop: 0 }}>
        <SettingOutlined /> 系统配置
      </Title>

      {meta?.env_overrides?.length > 0 && (
        <Alert
          type="warning"
          showIcon
          style={{ marginBottom: 16 }}
          message="以下项由环境变量 / .env 覆盖（修改 YAML 后可能不生效）"
          description={
            <Space wrap>
              {meta.env_overrides.map((k) => (
                <Tag key={k}>{k}</Tag>
              ))}
            </Space>
          }
        />
      )}

      {meta?.restart_note && (
        <Alert type="info" showIcon style={{ marginBottom: 16 }} message={meta.restart_note} />
      )}

      {meta?.config_path && (
        <Paragraph type="secondary">
          配置文件：<Text code>{meta.config_path}</Text>
          {meta.dotenv_path && (
            <>
              {' '}
              | .env：<Text code>{meta.dotenv_path}</Text>
            </>
          )}
        </Paragraph>
      )}

      <Form form={form} layout="vertical" disabled={!canUpdate}>
        <Tabs items={tabItems} />
      </Form>

      {canUpdate && (
        <>
          <Divider>更新密钥（留空表示不修改；显示 *** 表示已配置）</Divider>
          <Form form={secretsForm} layout="vertical" style={{ maxWidth: 480 }}>
            <Form.Item name="jwt_secret" label="JWT_SECRET">
              <Input.Password placeholder="留空不修改" autoComplete="new-password" />
            </Form.Item>
            <Form.Item name="aes_secret" label="AES_SECRET_KEY">
              <Input.Password placeholder="留空不修改" autoComplete="new-password" />
            </Form.Item>
            <Form.Item name="database_password" label="DB_PASSWORD">
              <Input.Password placeholder="留空不修改" autoComplete="new-password" />
            </Form.Item>
            <Form.Item name="redis_password" label="REDIS_PASSWORD">
              <Input.Password placeholder="留空不修改" autoComplete="new-password" />
            </Form.Item>
            <Form.Item name="email_password" label="EMAIL_PASSWORD">
              <Input.Password placeholder="留空不修改" autoComplete="new-password" />
            </Form.Item>
            <Form.Item name="metrics_bearer_token" label="METRICS_BEARER_TOKEN">
              <Input.Password placeholder="留空不修改" autoComplete="new-password" />
            </Form.Item>
          </Form>
        </>
      )}

      <Space style={{ marginTop: 16 }} wrap>
        <Button icon={<ReloadOutlined />} onClick={load} loading={loading}>
          刷新
        </Button>
        {canUpdate && (
          <>
            <Button type="primary" icon={<SaveOutlined />} onClick={onSave} loading={saving}>
              保存到 config.yaml
            </Button>
            <Button icon={<ThunderboltOutlined />} onClick={onHotReload}>
              热更新（IP 黑名单等）
            </Button>
          </>
        )}
      </Space>

      {meta?.effective?.security?.ip_blacklist && (
        <Alert
          style={{ marginTop: 16 }}
          type="success"
          message={`当前生效：IP 自动防护 ${meta.effective.security.ip_blacklist.enabled ? '已启用' : '已关闭'}`}
        />
      )}
    </div>
  );
}

export default SystemConfigManagement;
