import React, { useState, useEffect } from 'react';
import { Table, Button, Modal, Form, Input, Select, Switch, message, Popconfirm, Tag, Space, Card, Checkbox, Menu, Dropdown } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, SettingOutlined } from '@ant-design/icons';
import { getUsers, getUser, createUser, updateUser, deleteUser } from '../api';
import { toUserTimezone } from '../utils/datetime';

function UserManagement() {
  const [data, setData] = useState([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [modalTitle, setModalTitle] = useState('添加用户');
  const [editingId, setEditingId] = useState(null);
  const [form] = Form.useForm();
  const [filters, setFilters] = useState({
    username: '',
    department_id: '',
    role_id: ''
  });
  // 列显示控制
  const [visibleColumns, setVisibleColumns] = useState({
    id: true,
    username: true,
    email: true,
    department: true,
    role: true,
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
      const result = await getUsers(1, 100); // 暂时获取所有用户
      setData(result.data || []);
    } catch (error) {
      message.error(error || '获取用户列表失败');
    } finally {
      setLoading(false);
    }
  };

  const handleFilterChange = (values) => {
    setFilters({
      username: values.username || '',
      department_id: values.department_id || '',
      role_id: values.role_id || ''
    });
  };

  const handleAdd = () => {
    setModalTitle('添加用户');
    setEditingId(null);
    form.resetFields();
    form.setFieldsValue({ status: true });
    setModalVisible(true);
  };

  const handleEdit = async (record) => {
    setModalTitle('编辑用户');
    setEditingId(record.id);
    try {
      const result = await getUser(record.id);
      form.setFieldsValue({
        username: result.data.username,
        email: result.data.email,
        department_id: result.data.department_id,
        role_id: result.data.role_id,
        status: result.data.status,
      });
      setModalVisible(true);
    } catch (error) {
      message.error(error || '获取用户信息失败');
    }
  };

  const handleDelete = async (id) => {
    try {
      await deleteUser(id);
      message.success('删除成功');
      fetchData();
    } catch (error) {
      message.error(error || '删除失败');
    }
  };

  const handleSubmit = async (values) => {
    try {
      if (editingId) {
        await updateUser(editingId, values);
        message.success('更新成功');
      } else {
        await createUser(values);
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
      title: '用户名',
      dataIndex: 'username',
      key: 'username',
      sorter: (a, b) => a.username.localeCompare(b.username),
      visible: visibleColumns.username,
      // 关键列，不能隐藏
      fixed: true,
    },
    {
      title: '邮箱',
      dataIndex: 'email',
      key: 'email',
      sorter: (a, b) => a.email.localeCompare(b.email),
      visible: visibleColumns.email,
    },
    {
      title: '部门',
      dataIndex: 'department',
      key: 'department',
      render: (department) => department ? department.name : '-',
      sorter: (a, b) => {
        const deptA = a.department ? a.department.name : '';
        const deptB = b.department ? b.department.name : '';
        return deptA.localeCompare(deptB);
      },
      visible: visibleColumns.department,
    },
    {
      title: '角色',
      dataIndex: 'role',
      key: 'role',
      render: (role) => role ? role.name : '-',
      sorter: (a, b) => {
        const roleA = a.role ? a.role.name : '';
        const roleB = b.role ? b.role.name : '';
        return roleA.localeCompare(roleB);
      },
      visible: visibleColumns.role,
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
            title="确认删除此用户?"
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
        <h2 style={{ margin: 0 }}>用户管理</h2>
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
            添加用户
          </Button>
        </div>
      </div>

      <Card style={{ marginBottom: 24 }}>
        <Form
          layout="inline"
          onFinish={handleFilterChange}
          initialValues={filters}
        >
          <Form.Item name="username" label="用户名">
            <Input placeholder="请输入用户名" style={{ width: 150 }} />
          </Form.Item>
          <Form.Item name="department_id" label="部门">
            <Select placeholder="请选择部门" style={{ width: 150 }} allowClear>
              {/* 动态获取部门列表 */}
            </Select>
          </Form.Item>
          <Form.Item name="role_id" label="角色">
            <Select placeholder="请选择角色" style={{ width: 150 }} allowClear>
              {/* 动态获取角色列表 */}
            </Select>
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                筛选
              </Button>
              <Button 
                onClick={() => {
                  setFilters({ username: '', department_id: '', role_id: '' });
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
            name="username"
            label="用户名"
            rules={[{ required: true, message: '请输入用户名' }]}
          >
            <Input placeholder="请输入用户名" />
          </Form.Item>

          <Form.Item
            name="email"
            label="邮箱"
            rules={[{ required: true, message: '请输入邮箱' }]}
          >
            <Input placeholder="请输入邮箱" />
          </Form.Item>

          <Form.Item
            name="password"
            label="密码"
            rules={[editingId ? {} : { required: true, message: '请输入密码' }]}
          >
            <Input type="password" placeholder="请输入密码" />
          </Form.Item>

          <Form.Item
            name="department_id"
            label="部门"
          >
            <Select placeholder="请选择部门" allowClear>
              {/* 动态获取部门列表 */}
            </Select>
          </Form.Item>

          <Form.Item
            name="role_id"
            label="角色"
          >
            <Select placeholder="请选择角色" allowClear>
              {/* 动态获取角色列表 */}
            </Select>
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

export default UserManagement;