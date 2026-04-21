import React, { useState, useEffect, useCallback } from 'react';
import { Table, Button, Modal, Form, Input, message, Popconfirm, Tag, Space, Transfer, Card, Checkbox, Menu, Dropdown } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, SettingOutlined } from '@ant-design/icons';
import { getRoles, getRole, createRole, updateRole, deleteRole, getPermissions, assignPermissions } from '../api';
import { toUserTimezone } from '../utils/datetime';

function RoleManagement() {
  const [data, setData] = useState([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [permissionModalVisible, setPermissionModalVisible] = useState(false);
  const [modalTitle, setModalTitle] = useState('添加角色');
  const [editingId, setEditingId] = useState(null);
  const [allPermissions, setAllPermissions] = useState([]);
  const [selectedPermissions, setSelectedPermissions] = useState([]);
  const [currentRoleId, setCurrentRoleId] = useState(null);
  const [form] = Form.useForm();
  const [filters, setFilters] = useState({
    name: ''
  });
  // 列显示控制
  const [visibleColumns, setVisibleColumns] = useState({
    id: true,
    name: true,
    description: true,
    permissions: true,
    created_at: true,
    action: true
  });

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const result = await getRoles();
      
      // 应用筛选条件
      let filteredData = result.data || [];
      
      if (filters.name) {
        filteredData = filteredData.filter(role => 
          role.name.toLowerCase().includes(filters.name.toLowerCase())
        );
      }
      
      setData(filteredData);
    } catch (error) {
      message.error(error || '获取角色列表失败');
    } finally {
      setLoading(false);
    }
  }, [filters]);

  useEffect(() => {
    fetchData();
    fetchPermissions();
  }, [fetchData]);

  const handleFilterChange = (values) => {
    setFilters({
      name: values.name || ''
    });
  };

  const fetchPermissions = async () => {
    try {
      const result = await getPermissions();
      setAllPermissions(result.data || []);
    } catch (error) {
      console.error('获取权限列表失败:', error);
    }
  };

  const handleAdd = () => {
    setModalTitle('添加角色');
    setEditingId(null);
    form.resetFields();
    setModalVisible(true);
  };

  const handleEdit = async (record) => {
    setModalTitle('编辑角色');
    setEditingId(record.id);
    try {
      const result = await getRole(record.id);
      form.setFieldsValue({
        name: result.data.name,
        description: result.data.description,
      });
      setModalVisible(true);
    } catch (error) {
      message.error(error || '获取角色信息失败');
    }
  };

  const handleDelete = async (id) => {
    try {
      await deleteRole(id);
      message.success('删除成功');
      fetchData();
    } catch (error) {
      message.error(error || '删除失败');
    }
  };

  const handlePermission = async (record) => {
    setCurrentRoleId(record.id);
    try {
      const result = await getRole(record.id);
      const permissions = result.data.permissions?.map(p => p.id) || [];
      setSelectedPermissions(permissions);
      setPermissionModalVisible(true);
    } catch (error) {
      message.error(error || '获取角色权限失败');
    }
  };

  const handlePermissionSubmit = async () => {
    try {
      await assignPermissions(currentRoleId, selectedPermissions);
      message.success('权限分配成功');
      setPermissionModalVisible(false);
    } catch (error) {
      message.error(error || '权限分配失败');
    }
  };

  const handleSubmit = async (values) => {
    try {
      if (editingId) {
        await updateRole(editingId, values);
        message.success('更新成功');
      } else {
        await createRole(values);
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
      title: '角色名称',
      dataIndex: 'name',
      key: 'name',
      sorter: (a, b) => a.name.localeCompare(b.name),
      visible: visibleColumns.name,
      // 关键列，不能隐藏
      fixed: true,
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      sorter: (a, b) => {
        const descA = a.description || '';
        const descB = b.description || '';
        return descA.localeCompare(descB);
      },
      visible: visibleColumns.description,
    },
    {
      title: '权限',
      dataIndex: 'permissions',
      key: 'permissions',
      render: (permissions) => (
        <>
          {permissions && permissions.slice(0, 3).map((perm, index) => (
            <Tag key={index} color="green">{perm.name}</Tag>
          ))}
          {permissions && permissions.length > 3 && (
            <Tag color="default">+{permissions.length - 3}</Tag>
          )}
        </>
      ),
      sorter: (a, b) => {
        const permCountA = a.permissions ? a.permissions.length : 0;
        const permCountB = b.permissions ? b.permissions.length : 0;
        return permCountA - permCountB;
      },
      visible: visibleColumns.permissions,
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
            icon={<SettingOutlined />}
            onClick={() => handlePermission(record)}
          >
            权限
          </Button>
          <Button
            type="link"
            icon={<EditOutlined />}
            onClick={() => handleEdit(record)}
          >
            编辑
          </Button>
          <Popconfirm
            title="确认删除此角色?"
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

  const permissionData = allPermissions.map(perm => ({
    key: perm.id,
    title: perm.description || perm.name,
  }));

  return (
    <div>
      <div className="page-header" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <h2 style={{ margin: 0 }}>角色管理</h2>
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
            添加角色
          </Button>
        </div>
      </div>

      <Card style={{ marginBottom: 24 }}>
        <Form
          layout="inline"
          onFinish={handleFilterChange}
          initialValues={filters}
        >
          <Form.Item name="name" label="角色名称">
            <Input placeholder="请输入角色名称" style={{ width: 150 }} />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                筛选
              </Button>
              <Button 
                onClick={() => {
                  setFilters({ name: '' });
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
            label="角色名称"
            rules={[{ required: true, message: '请输入角色名称' }]}
          >
            <Input placeholder="请输入角色名称" />
          </Form.Item>

          <Form.Item
            name="description"
            label="描述"
          >
            <Input.TextArea placeholder="请输入描述" rows={3} />
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

      <Modal
        title="分配权限"
        open={permissionModalVisible}
        onCancel={() => setPermissionModalVisible(false)}
        onOk={handlePermissionSubmit}
        width={600}
      >
        <Transfer
          dataSource={permissionData}
          titles={['可选权限', '已选权限']}
          targetKeys={selectedPermissions}
          onChange={setSelectedPermissions}
          render={item => item.title}
          listStyle={{ width: 250, height: 300 }}
        />
      </Modal>
    </div>
  );
}

export default RoleManagement;