import React, { useState, useEffect, useCallback } from 'react';
import { Table, Button, Modal, Form, Input, Select, Switch, message, Popconfirm, Tag, Space, Card, Checkbox, Menu, Dropdown } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, SettingOutlined } from '@ant-design/icons';
import { getDepartments, getDepartment, createDepartment, updateDepartment, deleteDepartment } from '../api';
import { toUserTimezone } from '../utils/datetime';

function DepartmentManagement() {
  const [data, setData] = useState([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [modalTitle, setModalTitle] = useState('添加部门');
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
    description: true,
    parent: true,
    status: true,
    created_at: true,
    action: true
  });

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      console.log('开始获取部门数据...');
      // 当没有状态筛选时，获取所有部门（包括归档的）
      // 当有状态筛选时，只获取活跃部门，然后在前端进行筛选
      const onlyActive = filters.status !== '';
      const result = await getDepartments(onlyActive);
      console.log('获取到部门数据:', result);
      
      // 应用筛选条件
      let filteredData = result.data || [];
      
      if (filters.name) {
        filteredData = filteredData.filter(dept => 
          dept.name.toLowerCase().includes(filters.name.toLowerCase())
        );
      }
      
      if (filters.status) {
        const status = filters.status === 'true';
        filteredData = filteredData.filter(dept => dept.status === status);
      }
      
      setData(filteredData);
      console.log('设置数据后:', data);
    } catch (error) {
      console.error('获取部门列表失败:', error);
      message.error(error || '获取部门列表失败');
    } finally {
      setLoading(false);
      console.log('加载完成');
    }
  }, [filters]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const handleFilterChange = (values) => {
    setFilters({
      name: values.name || '',
      status: values.status || ''
    });
  };

  const buildTreeData = (departments) => {
    return departments.map(dept => ({
      key: dept.id,
      title: dept.name,
      children: dept.children && dept.children.length > 0 ? buildTreeData(dept.children) : undefined,
      ...dept,
    }));
  };

  const handleAdd = () => {
    setModalTitle('添加部门');
    setEditingId(null);
    form.resetFields();
    form.setFieldsValue({ status: true });
    setModalVisible(true);
  };

  const handleEdit = async (record) => {
    setModalTitle('编辑部门');
    setEditingId(record.id);
    try {
      const result = await getDepartment(record.id);
      form.setFieldsValue({
        name: result.data.name,
        description: result.data.description,
        parent_id: result.data.parent_id,
        status: result.data.status,
      });
      setModalVisible(true);
    } catch (error) {
      message.error(error || '获取部门信息失败');
    }
  };

  const handleDelete = async (id) => {
    try {
      await deleteDepartment(id);
      message.success('归档成功');
      fetchData();
    } catch (error) {
      message.error(error || '操作失败');
    }
  };

  const handleSubmit = async (values) => {
    try {
      if (editingId) {
        await updateDepartment(editingId, values);
        message.success('更新成功');
      } else {
        await createDepartment(values);
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
      title: '部门名称',
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
      title: '上级部门',
      dataIndex: 'parent',
      key: 'parent',
      render: (parent) => parent ? parent.name : '-',
      sorter: (a, b) => {
        const parentA = a.parent ? a.parent.name : '';
        const parentB = b.parent ? b.parent.name : '';
        return parentA.localeCompare(parentB);
      },
      visible: visibleColumns.parent,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status) => (
        <Tag color={status ? 'green' : 'red'}>
          {status ? '正常' : '归档'}
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
            title="确认归档此部门?"
            onConfirm={() => handleDelete(record.id)}
            okText="确认"
            cancelText="取消"
          >
            <Button type="link" danger icon={<DeleteOutlined />}>
              归档
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
        <h2 style={{ margin: 0 }}>部门管理</h2>
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
            添加部门
          </Button>
        </div>
      </div>

      <Card style={{ marginBottom: 24 }}>
        <Form
          layout="inline"
          onFinish={handleFilterChange}
          initialValues={filters}
        >
          <Form.Item name="name" label="部门名称">
            <Input placeholder="请输入部门名称" style={{ width: 150 }} />
          </Form.Item>
          <Form.Item name="status" label="状态">
            <Select placeholder="请选择状态" style={{ width: 100 }} allowClear>
              <Select.Option value="true">正常</Select.Option>
              <Select.Option value="false">归档</Select.Option>
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
        pagination={false}
      />
      {data.length === 0 && !loading && (
        <div style={{ textAlign: 'center', padding: '40px 0' }}>
          <p>暂无部门数据</p>
          <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
            添加部门
          </Button>
        </div>
      )}

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
            label="部门名称"
            rules={[{ required: true, message: '请输入部门名称' }]}
          >
            <Input placeholder="请输入部门名称" />
          </Form.Item>

          <Form.Item
            name="description"
            label="描述"
          >
            <Input.TextArea placeholder="请输入描述" rows={3} />
          </Form.Item>

          <Form.Item
            name="parent_id"
            label="上级部门"
          >
            <Select placeholder="请选择上级部门" allowClear>
              {data.filter(d => d.id !== editingId).map(dept => (
                <Select.Option key={dept.id} value={dept.id}>
                  {dept.name}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item
            name="status"
            label="状态"
            valuePropName="checked"
            initialValue={true}
          >
            <Switch checkedChildren="正常" unCheckedChildren="归档" />
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

export default DepartmentManagement;