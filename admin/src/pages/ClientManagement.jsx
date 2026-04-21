import React, { useState, useEffect } from 'react';
import { Table, Button, Modal, Form, Input, Select, Switch, message, Popconfirm, Tag, Space, Card, Checkbox, Menu, Dropdown } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, SettingOutlined } from '@ant-design/icons';
import { getClients, getClient, createClient, updateClient, deleteClient } from '../api';
import { toUserTimezone } from '../utils/datetime';

function ClientManagement() {
  const [data, setData] = useState([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [modalTitle, setModalTitle] = useState('添加客户');
  const [editingId, setEditingId] = useState(null);
  const [form] = Form.useForm();
  const [filters, setFilters] = useState({
    name: '',
    status: ''
  });
  // 列显示控制
  const [visibleColumns, setVisibleColumns] = useState({
    id: true,
    name: true,
    contact: true,
    email: true,
    status: true,
    created_at: true,
    action: true
  });

  // 只在 currentPage, pageSize 或 filters 变化时获取数据
  useEffect(() => {
    fetchData();
  }, [filters]);

  const fetchData = async () => {
    setLoading(true);
    try {
      const result = await getClients(1, 100); // 暂时获取所有客户
      setData(result.data || []);
    } catch (error) {
      message.error(error || '获取客户列表失败');
    } finally {
      setLoading(false);
    }
  };

  const handleFilterChange = (values) => {
    setFilters({
      name: values.name || '',
      status: values.status || ''
    });
  };

  const handleAdd = () => {
    setModalTitle('添加客户');
    setEditingId(null);
    form.resetFields();
    form.setFieldsValue({ status: true });
    setModalVisible(true);
  };

  const handleEdit = async (record) => {
    setModalTitle('编辑客户');
    setEditingId(record.id);
    try {
      const result = await getClient(record.id);
      form.setFieldsValue({
        name: result.data.name,
        contact: result.data.contact,
        email: result.data.email,
        status: result.data.status,
      });
      setModalVisible(true);
    } catch (error) {
      message.error(error || '获取客户信息失败');
    }
  };

  const handleDelete = async (id) => {
    try {
      await deleteClient(id);
      message.success('删除成功');
      fetchData();
    } catch (error) {
      message.error(error || '删除失败');
    }
  };

  const handleSubmit = async (values) => {
    try {
      if (editingId) {
        await updateClient(editingId, values);
        message.success('更新成功');
      } else {
        await createClient(values);
        message.success('创建成功');
      }
      setModalVisible(false);
      fetchData();
    } catch (error) {
      message.error(error || '操作失败');
    }
  };

  // 处理列显示/隐藏
  const handleColumnVisibilityChange = (changedColumns) => {
    setVisibleColumns({
      ...visibleColumns,
      ...changedColumns
    });
  };

  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      key: 'id',
      width: 60,
      sorter: (a, b) => a.id - b.id,
      defaultSortOrder: 'descend',
      visible: visibleColumns.id,
      // 关键列，不能隐藏
      fixed: true,
    },
    {
      title: '客户名称',
      dataIndex: 'name',
      key: 'name',
      sorter: (a, b) => a.name.localeCompare(b.name),
      visible: visibleColumns.name,
      // 关键列，不能隐藏
      fixed: true,
    },
    {
      title: '联系人',
      dataIndex: 'contact',
      key: 'contact',
      sorter: (a, b) => a.contact.localeCompare(b.contact),
      visible: visibleColumns.contact,
    },
    {
      title: '邮箱',
      dataIndex: 'email',
      key: 'email',
      sorter: (a, b) => a.email.localeCompare(b.email),
      visible: visibleColumns.email,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status) => (
        <Tag color={status ? 'green' : 'red'}>
          {status ? '正常' : '禁用'}
        </Tag>
      ),
      sorter: (a, b) => (a.status ? 1 : 0) - (b.status ? 1 : 0),
      visible: visibleColumns.status,
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      // 将 UTC 时间转换为用户时区显示
      render: (text) => toUserTimezone(text),
      sorter: (a, b) => new Date(a.created_at) - new Date(b.created_at),
      visible: visibleColumns.created_at,
    },
    {
      title: '操作',
      key: 'action',
      render: (_, record) => (
        <Space size="small">
          <Button
            type="link"
            icon={<EditOutlined />}
            onClick={() => handleEdit(record)}
          >
            编辑
          </Button>
          <Popconfirm
            title="确认删除此客户?"
            onConfirm={() => handleDelete(record.id)}
            okText="确认"
            cancelText="取消"
          >
            <Button type="link" danger icon={<DeleteOutlined />}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
      visible: visibleColumns.action,
      fixed: 'right',
    },
  ];

  return (
    <div>
      <div className="page-header" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <h2 style={{ margin: 0 }}>客户管理</h2>
        <div style={{ display: 'flex', alignItems: 'center' }}>
          {/* 列配置下拉菜单 */}
          <Dropdown
            overlay={
              <Menu>
                {columns.map(column => (
                  <Menu.Item key={column.key}>
                    <Checkbox
                      checked={column.visible}
                      onChange={(e) => {
                        // 关键列不能隐藏
                        if (column.fixed && !e.target.checked) {
                          message.warning('关键列不能隐藏');
                          return;
                        }
                        handleColumnVisibilityChange({ [column.key]: e.target.checked });
                      }}
                      disabled={column.fixed}
                    >
                      {column.title}
                    </Checkbox>
                  </Menu.Item>
                ))}
              </Menu>
            }
            trigger={['click']}
          >
            <Button icon={<SettingOutlined />} style={{ marginRight: 16 }}>
              列配置
            </Button>
          </Dropdown>
          <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
            添加客户
          </Button>
        </div>
      </div>

      <Card style={{ marginBottom: 24 }}>
        <Form
          layout="inline"
          onFinish={handleFilterChange}
          initialValues={filters}
        >
          <Form.Item name="name" label="客户名称">
            <Input placeholder="请输入客户名称" style={{ width: 150 }} />
          </Form.Item>
          <Form.Item name="status" label="状态">
            <Select placeholder="请选择状态" style={{ width: 100 }} allowClear>
              <Select.Option value="true">正常</Select.Option>
              <Select.Option value="false">禁用</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                筛选
              </Button>
              <Button 
                onClick={() => {
                  setFilters({ name: '', status: '' });
                }}
              >
                重置
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Card>

      <Table
        columns={columns.filter(col => col.visible)}
        dataSource={data}
        rowKey="id"
        loading={loading}
      />

      <Modal
        title={modalTitle}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        footer={null}
        width={500}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleSubmit}
        >
          <Form.Item
            name="name"
            label="客户名称"
            rules={[{ required: true, message: '请输入客户名称' }]}
          >
            <Input placeholder="请输入客户名称" />
          </Form.Item>

          <Form.Item
            name="contact"
            label="联系人"
            rules={[{ required: true, message: '请输入联系人' }]}
          >
            <Input placeholder="请输入联系人" />
          </Form.Item>

          <Form.Item
            name="email"
            label="邮箱"
            rules={[{ required: true, message: '请输入邮箱' }]}
          >
            <Input placeholder="请输入邮箱" />
          </Form.Item>

          <Form.Item
            name="status"
            label="状态"
            valuePropName="checked"
            initialValue={true}
          >
            <Switch checkedChildren="正常" unCheckedChildren="禁用" />
          </Form.Item>

          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                {editingId ? '更新' : '创建'}
              </Button>
              <Button onClick={() => setModalVisible(false)}>
                取消
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}

export default ClientManagement;