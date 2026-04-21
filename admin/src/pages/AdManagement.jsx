import React, { useState, useEffect } from 'react';
import { Table, Button, Modal, Form, Input, Select, message, Popconfirm, Tag, Space, DatePicker, InputNumber, Card, Checkbox, Descriptions, Menu, Dropdown } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, SyncOutlined, EyeOutlined, SettingOutlined } from '@ant-design/icons';
import { getAdTasks, getActiveAdTasks, getAdTask, createAdTask, updateAdTask, deleteAdTask, batchUpdateAdTasks } from '../api';
import { toUTC, toUserTimezone, toUserTimezoneDayjs } from '../utils/datetime';

const { TextArea } = Input;
const { RangePicker } = DatePicker;

function AdManagement() {
  const [data, setData] = useState([]);
  const [loading, setLoading] = useState(false);
  // 将 pagination 拆分为独立的 state，避免无限循环
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [total, setTotal] = useState(0);
  const [modalVisible, setModalVisible] = useState(false);
  const [modalTitle, setModalTitle] = useState('添加广告');
  const [editingId, setEditingId] = useState(null);
  const [form] = Form.useForm();
  const [filters, setFilters] = useState({
    name: '',
    status: '',
    time_range: null
  });
  const [selectedRowKeys, setSelectedRowKeys] = useState([]);
  const [useActiveView, setUseActiveView] = useState(false);
  const [detailModalVisible, setDetailModalVisible] = useState(false);
  const [detailData, setDetailData] = useState(null);
  const [sorter, setSorter] = useState({
    field: 'id',
    order: 'descend'
  });
  // 列显示控制
  const [visibleColumns, setVisibleColumns] = useState({
    id: true,
    name: true,
    client_id: true,
    status: true,
    plat: true,
    os: true,
    budget: true,
    action: true
  });

  // 将 time_range 序列化为字符串，避免对象引用变化导致的无限循环
  const timeRangeKey = filters.time_range 
    ? `${toUTC(filters.time_range[0])}_${toUTC(filters.time_range[1])}`
    : '';

  // 只在 currentPage, pageSize, filters, sorter 或 useActiveView 变化时获取数据
  useEffect(() => {
    fetchData();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [currentPage, pageSize, filters.name, filters.status, timeRangeKey, useActiveView, sorter.field, sorter.order]);

  const fetchData = async () => {
    setLoading(true);
    try {
      // 构建筛选参数
      const filterParams = {
        name: filters.name,
        status: filters.status
      };

      // 处理时间范围 - 将用户时区时间转换为 UTC
      if (filters.time_range && filters.time_range.length === 2) {
        filterParams.start_time = toUTC(filters.time_range[0]);
        filterParams.end_time = toUTC(filters.time_range[1]);
      }

      // 添加排序参数
      if (sorter.field && sorter.order) {
        filterParams.sort_field = sorter.field;
        filterParams.sort_order = sorter.order === 'ascend' ? 'asc' : 'desc';
      }

      // 根据是否使用活跃视图选择不同的API
      const result = useActiveView
        ? await getActiveAdTasks(currentPage, pageSize, filterParams)
        : await getAdTasks(currentPage, pageSize, filterParams);

      setData(result.data || []);
      // 只更新 total，不触发重新请求
      setTotal(result.meta?.total || 0);
    } catch (error) {
      message.error(error || '获取广告列表失败');
    } finally {
      setLoading(false);
    }
  };

  // 批量操作处理
  const handleBatchAction = async (action) => {
    if (selectedRowKeys.length === 0) {
      message.warning('请选择要操作的广告任务');
      return;
    }

    try {
      let updateData = {};
      switch (action) {
        case 'active':
          updateData = { status: 'active' };
          break;
        case 'paused':
          updateData = { status: 'paused' };
          break;
        case 'completed':
          updateData = { status: 'completed' };
          break;
        default:
          return;
      }

      await batchUpdateAdTasks(selectedRowKeys, updateData);
      message.success(`批量${action === 'active' ? '启用' : action === 'paused' ? '暂停' : '完成'}成功`);
      setSelectedRowKeys([]);
      fetchData();
    } catch (error) {
      message.error(error || '批量操作失败');
    }
  };

  const handleFilterChange = (values) => {
    setFilters({
      name: values.name || '',
      status: values.status || '',
      time_range: values.time_range || null
    });
  };

  const handleAdd = () => {
    setModalTitle('添加广告');
    setEditingId(null);
    form.resetFields();
    form.setFieldsValue({ status: 'active', priority: 0 });
    setModalVisible(true);
  };

  const handleEdit = async (record) => {
    setModalTitle('编辑广告');
    setEditingId(record.id);
    try {
      const result = await getAdTask(record.id);
      // 后端返回的是 UTC 标准时间，需要转换为用户时区显示
      const values = {
        ...result.data,
        // 设置 time_range 字段用于表单显示，将 UTC 转换为用户时区
        time_range: result.data.start_time && result.data.end_time
          ? [toUserTimezoneDayjs(result.data.start_time), toUserTimezoneDayjs(result.data.end_time)]
          : null,
      };
      form.setFieldsValue(values);
      setModalVisible(true);
    } catch (error) {
      message.error(error || '获取广告信息失败');
    }
  };

  const handleDetail = async (record) => {
    try {
      const result = await getAdTask(record.id);
      setDetailData(result.data);
      setDetailModalVisible(true);
    } catch (error) {
      message.error(error || '获取广告信息失败');
    }
  };

  const handleDelete = async (id) => {
    try {
      await deleteAdTask(id);
      message.success('删除成功');
      fetchData();
    } catch (error) {
      message.error(error || '删除失败');
    }
  };

  const handleSubmit = async (values) => {
    try {
      // 处理JSON格式字段
      const processJsonField = (field) => {
        if (!field) return null;
        try {
          return typeof field === 'string' ? JSON.parse(field) : field;
        } catch (e) {
          return field;
        }
      };

      const submitData = {
        ...values,
        // 将用户时区时间转换为 UTC 发送给后端
        start_time: values.time_range?.[0] ? toUTC(values.time_range[0]) : null,
        end_time: values.time_range?.[1] ? toUTC(values.time_range[1]) : null,
        target_audience: processJsonField(values.target_audience),
        creative: processJsonField(values.creative),
        intercept_info: processJsonField(values.intercept_info),
        allow_county: processJsonField(values.allow_county),
        final_domains: processJsonField(values.final_domains),
        info: processJsonField(values.info),
        user_quality: processJsonField(values.user_quality),
      };
      delete submitData.time_range;

      if (editingId) {
        await updateAdTask(editingId, submitData);
        message.success('更新成功');
      } else {
        await createAdTask(submitData);
        message.success('创建成功');
      }
      setModalVisible(false);
      fetchData();
    } catch (error) {
      message.error(error || '操作失败');
    }
  };

  const handleTableChange = (pag, filters, sorter) => {
    // 分别更新，避免创建新对象
    setCurrentPage(pag.current);
    setPageSize(pag.pageSize);
    
    // 处理排序逻辑
    if (sorter.field && sorter.order) {
      setSorter({
        field: sorter.field,
        order: sorter.order
      });
    }
  };

  // 处理列显示/隐藏
  const handleColumnVisibilityChange = (changedColumns) => {
    setVisibleColumns({
      ...visibleColumns,
      ...changedColumns
    });
  };

  const getStatusColor = (status) => {
    const colorMap = {
      active: 'green',
      paused: 'orange',
      completed: 'blue',
      expired: 'red',
    };
    return colorMap[status] || 'default';
  };

  const getStatusText = (status) => {
    const textMap = {
      active: '进行中',
      paused: '已暂停',
      completed: '已完成',
      expired: '已过期',
    };
    return textMap[status] || status;
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
      title: '广告名称',
      dataIndex: 'name',
      key: 'name',
      sorter: (a, b) => a.name.localeCompare(b.name),
      visible: visibleColumns.name,
      // 关键列，不能隐藏
      fixed: true,
    },
    {
      title: '客户ID',
      dataIndex: 'client_id',
      key: 'client_id',
      width: 80,
      sorter: (a, b) => a.client_id - b.client_id,
      visible: visibleColumns.client_id,
      // 关键列，不能隐藏
      fixed: true,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status) => (
        <Tag color={getStatusColor(status)}>{getStatusText(status)}</Tag>
      ),
      sorter: (a, b) => a.status.localeCompare(b.status),
      visible: visibleColumns.status,
    },
    {
      title: '平台',
      dataIndex: 'plat',
      key: 'plat',
      width: 100,
      sorter: (a, b) => a.plat.localeCompare(b.plat),
      visible: visibleColumns.plat,
    },
    {
      title: '操作系统',
      dataIndex: 'os',
      key: 'os',
      width: 100,
      render: (os) => os === 1 ? 'Android' : os === 2 ? 'iOS' : '-',
      sorter: (a, b) => a.os - b.os,
      visible: visibleColumns.os,
    },
    {
      title: '预算',
      dataIndex: 'budget',
      key: 'budget',
      render: (budget) => budget ? `¥${budget}` : '-',
      sorter: (a, b) => (a.budget || 0) - (b.budget || 0),
      visible: visibleColumns.budget,
    },
    {
      title: '操作',
      key: 'action',
      width: 200,
      render: (_, record) => (
        <Space size="small">
          <Button
            type="link"
            icon={<EditOutlined />}
            onClick={() => handleEdit(record)}
          >
            编辑
          </Button>
          <Button
            type="link"
            icon={<EyeOutlined />}
            onClick={() => handleDetail(record)}
          >
            详情
          </Button>
          <Popconfirm
            title="确认删除此广告?"
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
        <div style={{ display: 'flex', alignItems: 'center' }}>
          <h2 style={{ margin: 0, marginRight: 16 }}>广告管理</h2>
          <Checkbox
            checked={useActiveView}
            onChange={(e) => setUseActiveView(e.target.checked)}
            style={{ marginRight: 16 }}
          >
            使用活跃视图
          </Checkbox>
          <Button
            icon={<SyncOutlined />}
            onClick={fetchData}
            loading={loading}
            style={{ marginRight: 16 }}
          >
            刷新
          </Button>
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
            <Button icon={<SettingOutlined />}>
              列配置
            </Button>
          </Dropdown>
        </div>
        <div style={{ display: 'flex', alignItems: 'center' }}>
          {selectedRowKeys.length > 0 && (
            <Space style={{ marginRight: 16 }}>
              <Button onClick={() => handleBatchAction('active')}>批量启用</Button>
              <Button onClick={() => handleBatchAction('paused')}>批量暂停</Button>
              <Button onClick={() => handleBatchAction('completed')}>批量完成</Button>
            </Space>
          )}
          <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
            添加广告
          </Button>
        </div>
      </div>

      <Card style={{ marginBottom: 24 }}>
        <Form
          layout="inline"
          onFinish={handleFilterChange}
          initialValues={filters}
        >
          <Form.Item name="name" label="广告名称">
            <Input placeholder="请输入广告名称" style={{ width: 150 }} />
          </Form.Item>
          <Form.Item name="status" label="状态">
            <Select placeholder="请选择状态" style={{ width: 120 }} allowClear>
              <Select.Option value="active">进行中</Select.Option>
              <Select.Option value="paused">已暂停</Select.Option>
              <Select.Option value="completed">已完成</Select.Option>
              <Select.Option value="expired">已过期</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item name="time_range" label="投放时间">
            <RangePicker showTime style={{ width: 300 }} />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                筛选
              </Button>
              <Button
                onClick={() => {
                  setFilters({ name: '', status: '', time_range: null });
                  setCurrentPage(1);
                  setPageSize(10);
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
        pagination={{ current: currentPage, pageSize: pageSize, total: total, showSizeChanger: true, showQuickJumper: true }}
        onChange={handleTableChange}
        rowSelection={{
          selectedRowKeys,
          onChange: (keys) => setSelectedRowKeys(keys),
        }}
      />

      <Modal
        title={modalTitle}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        footer={null}
        width={700}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleSubmit}
        >
          <Form.Item
            name="name"
            label="广告名称"
            rules={[{ required: true, message: '请输入广告名称' }]}
          >
            <Input placeholder="请输入广告名称" />
          </Form.Item>

          <Form.Item
            name="description"
            label="描述"
          >
            <TextArea placeholder="请输入广告描述" rows={3} />
          </Form.Item>

          <Form.Item
            name="client_id"
            label="客户ID"
          >
            <InputNumber min={1} style={{ width: '100%' }} placeholder="请输入客户ID" />
          </Form.Item>

          <Form.Item
            name="status"
            label="状态"
            initialValue="active"
          >
            <Select placeholder="请选择状态">
              <Select.Option value="active">进行中</Select.Option>
              <Select.Option value="paused">已暂停</Select.Option>
              <Select.Option value="completed">已完成</Select.Option>
              <Select.Option value="expired">已过期</Select.Option>
            </Select>
          </Form.Item>

          <Form.Item
            name="priority"
            label="优先级"
            initialValue={0}
          >
            <InputNumber min={0} max={100} style={{ width: '100%' }} />
          </Form.Item>

          <Form.Item
            name="time_range"
            label="投放时间"
          >
            <RangePicker showTime style={{ width: '100%' }} />
          </Form.Item>

          <Form.Item
            name="target_audience"
            label="目标受众"
          >
            <Input.TextArea placeholder="请输入目标受众（JSON格式）" rows={3} />
          </Form.Item>

          <Form.Item
            name="creative"
            label="创意"
          >
            <Input.TextArea placeholder="请输入创意（JSON格式）" rows={3} />
          </Form.Item>

          <Form.Item
            name="budget"
            label="预算"
          >
            <InputNumber
              min={0}
              precision={2}
              style={{ width: '100%' }}
              placeholder="请输入预算金额"
              formatter={value => `¥ ${value}`.replace(/\B(?=(\d{3})+(?!\d))/g, ',')}
              parser={value => value.replace(/¥\s?|(,*)/g, '')}
            />
          </Form.Item>

          <Form.Item
            name="proxy_strategy"
            label="代理策略"
            initialValue={1}
          >
            <InputNumber min={0} style={{ width: '100%' }} />
          </Form.Item>

          <Form.Item
            name="biz_type"
            label="业务类型"
            initialValue={1}
          >
            <InputNumber min={0} style={{ width: '100%' }} />
          </Form.Item>

          <Form.Item
            name="profile"
            label="设备类型"
            initialValue="mobile"
          >
            <Select placeholder="请选择设备类型">
              <Select.Option value="mobile">移动端</Select.Option>
              <Select.Option value="desktop">桌面端</Select.Option>
            </Select>
          </Form.Item>

          <Form.Item
            name="plat"
            label="平台"
            initialValue="googlead"
          >
            <Input placeholder="请输入平台名称" />
          </Form.Item>

          <Form.Item
            name="cau"
            label="CAU"
            initialValue={0}
          >
            <InputNumber min={0} style={{ width: '100%' }} />
          </Form.Item>

          <Form.Item
            name="os"
            label="操作系统"
            initialValue={1}
          >
            <Select placeholder="请选择操作系统">
              <Select.Option value={1}>Android</Select.Option>
              <Select.Option value={2}>iOS</Select.Option>
            </Select>
          </Form.Item>

          <Form.Item
            name="intercept_info"
            label="拦截信息"
          >
            <Input.TextArea placeholder="请输入拦截信息（JSON格式）" rows={3} />
          </Form.Item>

          <Form.Item
            name="landurl"
            label="落地页URL"
          >
            <Input placeholder="请输入落地页URL" />
          </Form.Item>

          <Form.Item
            name="allow_county"
            label="允许国家"
          >
            <Input.TextArea placeholder="请输入允许国家（JSON格式）" rows={3} />
          </Form.Item>

          <Form.Item
            name="final_domains"
            label="最终域名"
          >
            <Input.TextArea placeholder="请输入最终域名（JSON格式）" rows={3} />
          </Form.Item>

          <Form.Item
            name="info"
            label="额外信息"
          >
            <Input.TextArea placeholder="请输入额外信息（JSON格式）" rows={3} />
          </Form.Item>

          <Form.Item
            name="user_quality"
            label="用户质量"
          >
            <Input.TextArea placeholder="请输入用户质量（JSON格式）" rows={3} />
          </Form.Item>

          <Form.Item
            name="click_home_offset_x"
            label="点击主页偏移X"
            initialValue={100}
          >
            <InputNumber style={{ width: '100%' }} />
          </Form.Item>

          <Form.Item
            name="click_home_offset_y"
            label="点击主页偏移Y"
            initialValue={100}
          >
            <InputNumber style={{ width: '100%' }} />
          </Form.Item>

          <Form.Item
            name="click_result_offset_x"
            label="点击结果偏移X"
            initialValue={100}
          >
            <InputNumber style={{ width: '100%' }} />
          </Form.Item>

          <Form.Item
            name="click_result_offset_y"
            label="点击结果偏移Y"
            initialValue={100}
          >
            <InputNumber style={{ width: '100%' }} />
          </Form.Item>

          <Form.Item
            name="f_key_api"
            label="API密钥"
          >
            <Input placeholder="请输入API密钥" />
          </Form.Item>

          <Form.Item
            name="plan_id"
            label="执行计划ID"
          >
            <InputNumber min={0} style={{ width: '100%' }} />
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

      {/* 详情模态框 */}
      <Modal
        title="广告详情"
        open={detailModalVisible}
        onCancel={() => setDetailModalVisible(false)}
        footer={[
          <Button key="close" onClick={() => setDetailModalVisible(false)}>
            关闭
          </Button>
        ]}
        width={800}
      >
        <div style={{ maxHeight: 600, overflowY: 'auto' }}>
          <Descriptions column={1} bordered>
            <Descriptions.Item label="ID">{detailData?.id}</Descriptions.Item>
            <Descriptions.Item label="广告名称">{detailData?.name}</Descriptions.Item>
            <Descriptions.Item label="描述">{detailData?.description}</Descriptions.Item>
            <Descriptions.Item label="客户ID">{detailData?.client_id}</Descriptions.Item>
            <Descriptions.Item label="状态">
              <Tag color={getStatusColor(detailData?.status)}>{getStatusText(detailData?.status)}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="优先级">{detailData?.priority}</Descriptions.Item>
            <Descriptions.Item label="开始时间">
              {toUserTimezone(detailData?.start_time)}
            </Descriptions.Item>
            <Descriptions.Item label="结束时间">
              {toUserTimezone(detailData?.end_time)}
            </Descriptions.Item>
            <Descriptions.Item label="目标受众">
              <pre style={{ whiteSpace: 'pre-wrap', margin: 0 }}>
                {JSON.stringify(detailData?.target_audience, null, 2)}
              </pre>
            </Descriptions.Item>
            <Descriptions.Item label="创意">
              <pre style={{ whiteSpace: 'pre-wrap', margin: 0 }}>
                {JSON.stringify(detailData?.creative, null, 2)}
              </pre>
            </Descriptions.Item>
            <Descriptions.Item label="预算">{detailData?.budget ? `¥${detailData.budget}` : '-'}</Descriptions.Item>
            <Descriptions.Item label="代理策略">{detailData?.proxy_strategy}</Descriptions.Item>
            <Descriptions.Item label="业务类型">{detailData?.biz_type}</Descriptions.Item>
            <Descriptions.Item label="设备类型">{detailData?.profile}</Descriptions.Item>
            <Descriptions.Item label="平台">{detailData?.plat}</Descriptions.Item>
            <Descriptions.Item label="CAU">{detailData?.cau}</Descriptions.Item>
            <Descriptions.Item label="操作系统">
              {detailData?.os === 1 ? 'Android' : detailData?.os === 2 ? 'iOS' : '-'}
            </Descriptions.Item>
            <Descriptions.Item label="拦截信息">
              <pre style={{ whiteSpace: 'pre-wrap', margin: 0 }}>
                {JSON.stringify(detailData?.intercept_info, null, 2)}
              </pre>
            </Descriptions.Item>
            <Descriptions.Item label="落地页URL">{detailData?.landurl}</Descriptions.Item>
            <Descriptions.Item label="允许国家">
              <pre style={{ whiteSpace: 'pre-wrap', margin: 0 }}>
                {JSON.stringify(detailData?.allow_county, null, 2)}
              </pre>
            </Descriptions.Item>
            <Descriptions.Item label="最终域名">
              <pre style={{ whiteSpace: 'pre-wrap', margin: 0 }}>
                {JSON.stringify(detailData?.final_domains, null, 2)}
              </pre>
            </Descriptions.Item>
            <Descriptions.Item label="额外信息">
              <pre style={{ whiteSpace: 'pre-wrap', margin: 0 }}>
                {JSON.stringify(detailData?.info, null, 2)}
              </pre>
            </Descriptions.Item>
            <Descriptions.Item label="用户质量">
              <pre style={{ whiteSpace: 'pre-wrap', margin: 0 }}>
                {JSON.stringify(detailData?.user_quality, null, 2)}
              </pre>
            </Descriptions.Item>
            <Descriptions.Item label="点击主页偏移X">{detailData?.click_home_offset_x}</Descriptions.Item>
            <Descriptions.Item label="点击主页偏移Y">{detailData?.click_home_offset_y}</Descriptions.Item>
            <Descriptions.Item label="点击结果偏移X">{detailData?.click_result_offset_x}</Descriptions.Item>
            <Descriptions.Item label="点击结果偏移Y">{detailData?.click_result_offset_y}</Descriptions.Item>
            <Descriptions.Item label="API密钥">{detailData?.f_key_api}</Descriptions.Item>
            <Descriptions.Item label="执行计划ID">{detailData?.plan_id}</Descriptions.Item>
            <Descriptions.Item label="创建时间">
              {toUserTimezone(detailData?.created_at)}
            </Descriptions.Item>
            <Descriptions.Item label="更新时间">
              {toUserTimezone(detailData?.updated_at)}
            </Descriptions.Item>
            <Descriptions.Item label="最后访问时间">
              {toUserTimezone(detailData?.last_accessed_at)}
            </Descriptions.Item>
            <Descriptions.Item label="访问次数">{detailData?.access_count}</Descriptions.Item>
          </Descriptions>
        </div>
      </Modal>
    </div>
  );
}

export default AdManagement;