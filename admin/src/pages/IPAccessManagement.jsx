import React, { useState, useEffect, useCallback } from 'react';
import {
  Tabs,
  Table,
  Button,
  Modal,
  Form,
  Input,
  message,
  Popconfirm,
  Tag,
  Space,
  Alert,
  Typography,
} from 'antd';
import {
  PlusOutlined,
  DeleteOutlined,
  ReloadOutlined,
  SafetyCertificateOutlined,
  StopOutlined,
} from '@ant-design/icons';
import {
  getIPAccessStatus,
  getIPBlacklist,
  addIPBlacklist,
  removeIPBlacklist,
  getIPWhitelist,
  addIPWhitelist,
  removeIPWhitelist,
} from '../api';
import { toUserTimezone } from '../utils/datetime';
import usePermission from '../hooks/usePermission';

const { Text } = Typography;

const sourceLabels = {
  auto: '自动封禁',
  manual: '手动封禁',
  static_config: '配置静态',
};

function IPAccessManagement() {
  const { can } = usePermission();
  const canUpdate = can('user', 'update');

  const [status, setStatus] = useState({ enabled: false });
  const [blacklist, setBlacklist] = useState([]);
  const [whitelist, setWhitelist] = useState([]);
  const [loadingBlack, setLoadingBlack] = useState(false);
  const [loadingWhite, setLoadingWhite] = useState(false);
  const [blackModalOpen, setBlackModalOpen] = useState(false);
  const [whiteModalOpen, setWhiteModalOpen] = useState(false);
  const [blackForm] = Form.useForm();
  const [whiteForm] = Form.useForm();

  const fetchStatus = useCallback(async () => {
    try {
      const res = await getIPAccessStatus();
      setStatus(res.data || { enabled: false });
    } catch {
      setStatus({ enabled: false });
    }
  }, []);

  const fetchBlacklist = useCallback(async () => {
    setLoadingBlack(true);
    try {
      const res = await getIPBlacklist();
      setBlacklist(res.data?.entries || []);
    } catch (error) {
      message.error(error || '获取黑名单失败');
    } finally {
      setLoadingBlack(false);
    }
  }, []);

  const fetchWhitelist = useCallback(async () => {
    setLoadingWhite(true);
    try {
      const res = await getIPWhitelist();
      setWhitelist(res.data?.entries || []);
    } catch (error) {
      message.error(error || '获取白名单失败');
    } finally {
      setLoadingWhite(false);
    }
  }, []);

  useEffect(() => {
    fetchStatus();
    fetchBlacklist();
    fetchWhitelist();
  }, [fetchStatus, fetchBlacklist, fetchWhitelist]);

  const handleAddBlacklist = async () => {
    try {
      const values = await blackForm.validateFields();
      await addIPBlacklist(values);
      message.success('已加入黑名单');
      setBlackModalOpen(false);
      blackForm.resetFields();
      fetchBlacklist();
    } catch (error) {
      if (error?.errorFields) return;
      message.error(error || '添加失败');
    }
  };

  const handleAddWhitelist = async () => {
    try {
      const values = await whiteForm.validateFields();
      await addIPWhitelist(values);
      message.success('已加入白名单');
      setWhiteModalOpen(false);
      whiteForm.resetFields();
      fetchWhitelist();
      fetchBlacklist();
    } catch (error) {
      if (error?.errorFields) return;
      message.error(error || '添加失败');
    }
  };

  const handleRemoveBlacklist = async (ip) => {
    try {
      await removeIPBlacklist(ip);
      message.success('已解除封禁');
      fetchBlacklist();
    } catch (error) {
      message.error(error || '解除失败');
    }
  };

  const handleRemoveWhitelist = async (ip) => {
    try {
      await removeIPWhitelist(ip);
      message.success('已移出白名单');
      fetchWhitelist();
    } catch (error) {
      message.error(error || '移除失败');
    }
  };

  const renderSourceTag = (source, staticFlag) => {
    if (staticFlag) {
      return <Tag color="purple">配置静态</Tag>;
    }
    const label = sourceLabels[source] || source || '—';
    const color = source === 'auto' ? 'orange' : source === 'manual' ? 'blue' : 'default';
    return <Tag color={color}>{label}</Tag>;
  };

  const blacklistColumns = [
    {
      title: 'IP 地址',
      dataIndex: 'ip',
      key: 'ip',
      render: (ip) => <Text code>{ip}</Text>,
    },
    {
      title: '原因',
      dataIndex: 'reason',
      key: 'reason',
      ellipsis: true,
    },
    {
      title: '来源',
      dataIndex: 'source',
      key: 'source',
      render: (source, record) => renderSourceTag(source, record.static),
    },
    {
      title: '封禁时间',
      dataIndex: 'banned_at',
      key: 'banned_at',
      render: (t) => (t ? toUserTimezone(t) : '—'),
    },
    {
      title: '到期时间',
      key: 'expires',
      render: (_, record) => {
        if (record.permanent) return <Tag>永久</Tag>;
        if (record.expires_at) return toUserTimezone(record.expires_at);
        return '—';
      },
    },
    {
      title: '操作',
      key: 'action',
      width: 120,
      render: (_, record) =>
        canUpdate && !record.static ? (
          <Popconfirm
            title="确认解除该 IP 的封禁？"
            onConfirm={() => handleRemoveBlacklist(record.ip)}
          >
            <Button type="link" danger icon={<DeleteOutlined />} size="small">
              解封
            </Button>
          </Popconfirm>
        ) : record.static ? (
          <Text type="secondary">配置项</Text>
        ) : null,
    },
  ];

  const whitelistColumns = [
    {
      title: 'IP 地址',
      dataIndex: 'ip',
      key: 'ip',
      render: (ip) => <Text code>{ip}</Text>,
    },
    {
      title: '备注',
      dataIndex: 'remark',
      key: 'remark',
      ellipsis: true,
    },
    {
      title: '来源',
      dataIndex: 'source',
      key: 'source',
      render: (source, record) => renderSourceTag(source, record.static),
    },
    {
      title: '加入时间',
      dataIndex: 'added_at',
      key: 'added_at',
      render: (t) => (t ? toUserTimezone(t) : '—'),
    },
    {
      title: '操作',
      key: 'action',
      width: 120,
      render: (_, record) =>
        canUpdate && !record.static ? (
          <Popconfirm
            title="确认从白名单移除？"
            onConfirm={() => handleRemoveWhitelist(record.ip)}
          >
            <Button type="link" danger icon={<DeleteOutlined />} size="small">
              移除
            </Button>
          </Popconfirm>
        ) : record.static ? (
          <Text type="secondary">配置项</Text>
        ) : null,
    },
  ];

  const tabItems = [
    {
      key: 'blacklist',
      label: (
        <span>
          <StopOutlined /> 黑名单 ({blacklist.length})
        </span>
      ),
      children: (
        <>
          <Space style={{ marginBottom: 16 }} wrap>
            {canUpdate && (
              <Button type="primary" icon={<PlusOutlined />} onClick={() => setBlackModalOpen(true)}>
                手动封禁 IP
              </Button>
            )}
            <Button icon={<ReloadOutlined />} onClick={fetchBlacklist}>
              刷新
            </Button>
          </Space>
          <Table
            rowKey="ip"
            columns={blacklistColumns}
            dataSource={blacklist}
            loading={loadingBlack}
            pagination={{ pageSize: 10, showSizeChanger: true }}
            locale={{ emptyText: '暂无封禁记录' }}
          />
        </>
      ),
    },
    {
      key: 'whitelist',
      label: (
        <span>
          <SafetyCertificateOutlined /> 白名单 ({whitelist.length})
        </span>
      ),
      children: (
        <>
          <Space style={{ marginBottom: 16 }} wrap>
            {canUpdate && (
              <Button type="primary" icon={<PlusOutlined />} onClick={() => setWhiteModalOpen(true)}>
                添加白名单
              </Button>
            )}
            <Button icon={<ReloadOutlined />} onClick={fetchWhitelist}>
              刷新
            </Button>
          </Space>
          <Table
            rowKey="ip"
            columns={whitelistColumns}
            dataSource={whitelist}
            loading={loadingWhite}
            pagination={{ pageSize: 10, showSizeChanger: true }}
            locale={{ emptyText: '暂无白名单记录' }}
          />
        </>
      ),
    },
  ];

  return (
    <div>
      <Typography.Title level={4} style={{ marginTop: 0 }}>
        IP 访问控制
      </Typography.Title>
      <Alert
        type={status.enabled ? 'success' : 'warning'}
        showIcon
        style={{ marginBottom: 16 }}
        message={
          status.enabled
            ? '自动防护已开启：敏感路径异常访问将累计 strike 并可能自动封禁。白名单 IP 不会被封禁。'
            : '自动防护未开启（config 中 ip_blacklist.enabled=false），仍可在此管理黑白名单，启用配置后生效。'
        }
      />
      <Tabs items={tabItems} />

      <Modal
        title="手动封禁 IP"
        open={blackModalOpen}
        onOk={handleAddBlacklist}
        onCancel={() => setBlackModalOpen(false)}
        destroyOnClose
      >
        <Form form={blackForm} layout="vertical">
          <Form.Item
            name="ip"
            label="IP 地址"
            rules={[{ required: true, message: '请输入 IP 地址' }]}
          >
            <Input placeholder="例如 203.0.113.1 或 2001:db8::1" />
          </Form.Item>
          <Form.Item name="reason" label="封禁原因">
            <Input.TextArea rows={2} placeholder="可选，如：恶意扫描" />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title="添加白名单"
        open={whiteModalOpen}
        onOk={handleAddWhitelist}
        onCancel={() => setWhiteModalOpen(false)}
        destroyOnClose
      >
        <Form form={whiteForm} layout="vertical">
          <Form.Item
            name="ip"
            label="IP 地址"
            rules={[{ required: true, message: '请输入 IP 地址' }]}
          >
            <Input placeholder="例如 203.0.113.1" />
          </Form.Item>
          <Form.Item name="remark" label="备注">
            <Input.TextArea rows={2} placeholder="可选，如：办公出口 IP" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}

export default IPAccessManagement;
