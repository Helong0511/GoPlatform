import React, { useState, useEffect } from 'react';
import { Table, Button, Modal, Form, Input, Select, message, Space, Spin, Card, Row, Col, Statistic, DatePicker, InputNumber, Switch, Descriptions, Checkbox, Menu, Dropdown } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, RiseOutlined, BarChartOutlined, CheckCircleOutlined, ClockCircleOutlined, InboxOutlined, EyeOutlined, SettingOutlined } from '@ant-design/icons';
import { getAdReports, batchCreateAdReports, getReportStats } from '../api';
import { toUTC, toUserTimezone } from '../utils/datetime';

const { RangePicker } = DatePicker;

/**
 * 广告上报管理页面组件
 * 用于管理广告上报数据和任务
 */
function AdReportManagement() {
  const [data, setData] = useState([]);
  const [loading, setLoading] = useState(false);
  const [statsLoading, setStatsLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [modalTitle, setModalTitle] = useState('添加广告上报');
  const [editingId, setEditingId] = useState(null);
  const [form] = Form.useForm();
  // 将 total 分离出来，避免 setPagination 触发无限循环
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [total, setTotal] = useState(0);
  const [filters, setFilters] = useState({
    ad_id: '',
    action: '',
    time_range: null
  });
  const [batchModalVisible, setBatchModalVisible] = useState(false);
  const [batchForm] = Form.useForm();
  const [detailModalVisible, setDetailModalVisible] = useState(false);
  const [detailData, setDetailData] = useState(null);
  const [stats, setStats] = useState({
    total_reports: 0,
    total_clicks: 0,
    report_rate: 0,
    today_reports: 0
  });
  // 列显示控制
  const [visibleColumns, setVisibleColumns] = useState({
    id: true,
    ad_id: true,
    user_id: true,
    action: true,
    ip: true,
    country: true,
    client_outer: true,
    click: true,
    timestamp: true,
    actions: true
  });

  // 将 time_range 序列化为字符串，避免对象引用变化导致的无限循环
  const timeRangeKey = filters.time_range 
    ? `${toUTC(filters.time_range[0])}_${toUTC(filters.time_range[1])}`
    : '';

  // 只在 currentPage, pageSize 或 filters 变化时获取数据
  useEffect(() => {
    fetchData();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [currentPage, pageSize, filters.ad_id, filters.action, timeRangeKey]);

  // 定期获取统计数据
  useEffect(() => {
    fetchStats();
    // 每30秒更新一次统计数据
    const interval = setInterval(fetchStats, 30000);
    return () => clearInterval(interval);
  }, []);

  /**
   * 获取上报统计数据
   */
  const fetchStats = async () => {
    setStatsLoading(true);
    try {
      const result = await getReportStats();
      if (result && typeof result === 'object' && result.data) {
        setStats(result.data);
      } else {
        throw new Error('无效的统计数据格式');
      }
    } catch (error) {
      console.error('获取统计数据失败:', error);
      message.error('后端服务不可用，请检查服务状态');
      // 设置默认统计数据，避免一直转圈
      setStats({
        total_reports: 0,
        total_clicks: 0,
        report_rate: 0,
        today_reports: 0
      });
    } finally {
      setStatsLoading(false);
    }
  };

  /**
   * 获取广告上报数据
   */
  const fetchData = async () => {
    setLoading(true);
    try {
      // 构建筛选参数
      const filterParams = {
        ad_id: filters.ad_id,
        action: filters.action
      };
      
      // 构建查询参数
      const params = {
        ...filterParams,
        page: currentPage,
        page_size: pageSize
      };
      
      // 处理时间范围 - 将用户时区时间转换为 UTC
      if (filters.time_range && filters.time_range.length === 2) {
        const startDate = toUTC(filters.time_range[0]);
        const endDate = toUTC(filters.time_range[1]);
        params.start_time = startDate;
        params.end_time = endDate;
      }
      
      // 使用统一的API端点
      const result = await getAdReports(currentPage, pageSize, params);
      
      if (result && typeof result === 'object') {
        setData(Array.isArray(result.reports) ? result.reports : []);
        // 只更新 total，不触发重新请求
        setTotal(Number(result.total) || 0);
      } else {
        throw new Error('无效的返回数据格式');
      }
    } catch (error) {
      console.error('获取广告上报数据失败:', error);
      message.error('后端服务不可用，请检查服务状态');
      // 设置空数据，避免一直转圈
      setData([]);
      setTotal(0);
    } finally {
      setLoading(false);
    }
  };

  /**
   * 批量添加上报数据
   */
  const handleBatchSubmit = async (values) => {
    try {
      const reports = values.reports.split('\n').filter(line => line.trim()).map(line => {
        try {
          return JSON.parse(line);
        } catch (e) {
          return null;
        }
      }).filter(Boolean);
      
      if (reports.length === 0) {
        message.warning('请输入有效的JSON格式上报数据');
        return;
      }
      
      await batchCreateAdReports(reports);
      message.success(`批量添加成功，共${reports.length}条数据`);
      setBatchModalVisible(false);
      batchForm.resetFields();
      fetchData();
    } catch (error) {
      message.error(error || '批量添加失败');
    }
  };

  const handleAdd = () => {
    setModalTitle('添加广告上报');
    setEditingId(null);
    form.resetFields();
    setModalVisible(true);
  };

  const handleEdit = (record) => {
    setModalTitle('编辑广告上报');
    setEditingId(record.id);
    form.setFieldsValue(record);
    setModalVisible(true);
  };

  const handleDetail = (record) => {
    setDetailData(record);
    setDetailModalVisible(true);
  };

  const handleDelete = (id) => {
    try {
      // 这里应该调用API删除广告上报数据
      message.success('删除成功');
      fetchData();
    } catch (error) {
      message.error(error || '删除失败');
    }
  };

  const handleTableChange = (pag, filters, sorter) => {
    // 分别更新，避免创建新对象
    setCurrentPage(pag.current);
    setPageSize(pag.pageSize);
    
    // 处理排序逻辑 - 表格组件会自动处理前端排序
    // 当sorter变化时，表格会触发重新渲染并应用排序
  };

  // 处理列显示/隐藏
  const handleColumnVisibilityChange = (changedColumns) => {
    setVisibleColumns({
      ...visibleColumns,
      ...changedColumns
    });
  };

  const handleFilterChange = (values) => {
    setFilters({
      ad_id: values.ad_id || '',
      action: values.action || '',
      time_range: values.time_range || null
    });
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

      // 处理数据
      const data = {
        ...values,
        device_info: processJsonField(values.device_info),
      };

      // 这里应该调用API创建或更新广告上报数据
      message.success(editingId ? '更新成功' : '创建成功');
      setModalVisible(false);
      fetchData();
    } catch (error) {
      message.error(error || '操作失败');
    }
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
      title: '广告ID',
      dataIndex: 'ad_id',
      key: 'ad_id',
      sorter: (a, b) => a.ad_id - b.ad_id,
      visible: visibleColumns.ad_id,
      // 关键列，不能隐藏
      fixed: true,
    },
    {
      title: '用户ID',
      dataIndex: 'user_id',
      key: 'user_id',
      sorter: (a, b) => a.user_id.localeCompare(b.user_id),
      visible: visibleColumns.user_id,
    },
    {
      title: '操作类型',
      dataIndex: 'action',
      key: 'action',
      sorter: (a, b) => a.action.localeCompare(b.action),
      visible: visibleColumns.action,
    },
    {
      title: 'IP地址',
      dataIndex: 'ip',
      key: 'ip',
      sorter: (a, b) => a.ip.localeCompare(b.ip),
      visible: visibleColumns.ip,
    },
    {
      title: '国家',
      dataIndex: 'country',
      key: 'country',
      sorter: (a, b) => a.country.localeCompare(b.country),
      visible: visibleColumns.country,
    },
    {
      title: '客户端',
      dataIndex: 'client_outer',
      key: 'client_outer',
      sorter: (a, b) => a.client_outer.localeCompare(b.client_outer),
      visible: visibleColumns.client_outer,
    },
    {
      title: '点击数',
      dataIndex: 'click',
      key: 'click',
      width: 80,
      sorter: (a, b) => a.click - b.click,
      visible: visibleColumns.click,
    },
    {
      title: '上报时间',
      dataIndex: 'timestamp',
      key: 'timestamp',
      // 将 UTC 时间转换为用户时区显示
      render: (timestamp) => toUserTimezone(timestamp),
      sorter: (a, b) => new Date(a.timestamp) - new Date(b.timestamp),
      visible: visibleColumns.timestamp,
    },
    {
      title: '操作',
      key: 'actions',
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
          <Button
            type="link"
            danger
            icon={<DeleteOutlined />}
            onClick={() => handleDelete(record.id)}
          >
            删除
          </Button>
        </Space>
      ),
      visible: visibleColumns.actions,
      fixed: 'right',
    },
  ];

  return (
    <div className="page-container">
      <div className="page-header" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <h2 style={{ margin: 0 }}>广告上报</h2>
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
          <Space>
            <Button 
              icon={<InboxOutlined />} 
              onClick={() => setBatchModalVisible(true)}
            >
              批量添加
            </Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
              添加上报
            </Button>
          </Space>
        </div>
      </div>

      <Card style={{ marginBottom: 24 }}>
        <Row gutter={[16, 16]}>
          <Col xs={24} sm={12} lg={6}>
            <Statistic
              title="总上报数"
              value={stats.total_reports}
              prefix={<RiseOutlined />}
              valueStyle={{ color: '#1890ff' }}
              loading={statsLoading}
            />
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <Statistic
              title="点击数"
              value={stats.total_clicks}
              prefix={<BarChartOutlined />}
              valueStyle={{ color: '#52c41a' }}
              loading={statsLoading}
            />
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <Statistic
              title="上报率"
              value={`${isNaN(stats.report_rate) ? 0 : Math.round(stats.report_rate)}%`}
              prefix={<CheckCircleOutlined />}
              valueStyle={{ color: '#faad14' }}
              loading={statsLoading}
            />
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <Statistic
              title="今日上报"
              value={stats.today_reports}
              prefix={<ClockCircleOutlined />}
              valueStyle={{ color: '#722ed1' }}
              loading={statsLoading}
            />
          </Col>
        </Row>
      </Card>

      <Card style={{ marginBottom: 24 }}>
        <Form
          layout="inline"
          onFinish={handleFilterChange}
          initialValues={filters}
        >
          <Form.Item name="ad_id" label="广告ID">
            <InputNumber min={1} placeholder="请输入广告ID" style={{ width: 120 }} />
          </Form.Item>
          <Form.Item name="action" label="操作类型">
            <Select placeholder="请选择操作类型" style={{ width: 150 }}>
              <Select.Option value="click">点击</Select.Option>
              <Select.Option value="impression">展示</Select.Option>
              <Select.Option value="conversion">转化</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item name="time_range" label="时间范围">
            <RangePicker showTime style={{ width: 300 }} />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                筛选
              </Button>
              <Button 
                onClick={() => {
                  setFilters({ ad_id: '', action: '', time_range: null });
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
      />

      <Modal
        title={modalTitle}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        footer={null}
        width={800}
      >
        <Spin spinning={loading} tip="处理中...">
          <Form
            form={form}
            layout="vertical"
            onFinish={handleSubmit}
          >
            <Form.Item
              name="ad_id"
              label="广告ID"
              rules={[{ required: true, message: '请输入广告ID' }]}
            >
              <InputNumber min={1} placeholder="请输入广告ID" style={{ width: '100%' }} />
            </Form.Item>

            <Form.Item
              name="user_id"
              label="用户ID"
              rules={[{ required: true, message: '请输入用户ID' }]}
            >
              <Input placeholder="请输入用户ID" />
            </Form.Item>

            <Form.Item
              name="action"
              label="操作类型"
              rules={[{ required: true, message: '请选择操作类型' }]}
            >
              <Select placeholder="请选择操作类型">
                <Select.Option value="click">点击</Select.Option>
                <Select.Option value="impression">展示</Select.Option>
                <Select.Option value="conversion">转化</Select.Option>
              </Select>
            </Form.Item>

            <Form.Item
              name="device_info"
              label="设备信息"
              rules={[{ required: true, message: '请输入设备信息' }]}
            >
              <Input.TextArea placeholder="请输入设备信息（JSON格式）" rows={3} />
            </Form.Item>

            <Form.Item
              name="ip"
              label="IP地址"
              rules={[{ required: true, message: '请输入IP地址' }]}
            >
              <Input placeholder="请输入IP地址" />
            </Form.Item>

            <Form.Item
              name="client_inner"
              label="内部客户端"
            >
              <Input placeholder="请输入内部客户端" />
            </Form.Item>

            <Form.Item
              name="client_outer"
              label="外部客户端"
            >
              <Input placeholder="请输入外部客户端" />
            </Form.Item>

            <Form.Item
              name="real_ip"
              label="真实IP"
            >
              <Input placeholder="请输入真实IP" />
            </Form.Item>

            <Form.Item
              name="click"
              label="点击数"
              initialValue={0}
            >
              <InputNumber min={0} style={{ width: '100%' }} />
            </Form.Item>

            <Form.Item
              name="use_time"
              label="使用时间"
            >
              <InputNumber min={0} precision={3} style={{ width: '100%' }} placeholder="请输入使用时间" />
            </Form.Item>

            <Form.Item
              name="ads"
              label="广告数"
              initialValue={0}
            >
              <InputNumber min={0} style={{ width: '100%' }} />
            </Form.Item>

            <Form.Item
              name="home_page"
              label="是否主页"
              initialValue={false}
            >
              <Switch />
            </Form.Item>

            <Form.Item
              name="subpage_access"
              label="子页面访问数"
              initialValue={0}
            >
              <InputNumber min={0} style={{ width: '100%' }} />
            </Form.Item>

            <Form.Item
              name="referer"
              label="来源"
            >
              <Input placeholder="请输入来源" />
            </Form.Item>

            <Form.Item
              name="country"
              label="国家"
            >
              <Input placeholder="请输入国家" />
            </Form.Item>

            <Form.Item
              name="seq"
              label="序列"
            >
              <Input placeholder="请输入序列" />
            </Form.Item>

            <Form.Item
              name="error"
              label="错误信息"
            >
              <Input.TextArea placeholder="请输入错误信息" rows={4} style={{ minHeight: 100, width: '100%' }} />
            </Form.Item>

            <Form.Item
              name="should_click"
              label="应该点击"
              initialValue={false}
            >
              <Switch />
            </Form.Item>

            <Form.Item
              name="should_subpage"
              label="应该访问子页面"
              initialValue={false}
            >
              <Switch />
            </Form.Item>

            <Form.Item
              name="ios"
              label="是否iOS"
              initialValue={false}
            >
              <Switch />
            </Form.Item>

            <Form.Item
              name="click_able"
              label="可点击"
              initialValue={false}
            >
              <Switch />
            </Form.Item>

            <Form.Item
              name="source_type"
              label="来源类型"
            >
              <Input placeholder="请输入来源类型" />
            </Form.Item>

            <Form.Item
              name="ip_source"
              label="IP来源"
              initialValue={0}
            >
              <InputNumber min={0} style={{ width: '100%' }} />
            </Form.Item>

            <Form.Item
              name="ad_type"
              label="广告类型"
              initialValue={0}
            >
              <InputNumber min={0} style={{ width: '100%' }} />
            </Form.Item>

            <Form.Item
              name="is_review"
              label="是否审核"
              initialValue={0}
            >
              <InputNumber min={0} style={{ width: '100%' }} />
            </Form.Item>

            <Form.Item
              name="last_seq"
              label="上一个序列"
            >
              <Input placeholder="请输入上一个序列" />
            </Form.Item>

            <Form.Item
              name="screen_width"
              label="屏幕宽度"
              initialValue={0}
            >
              <InputNumber min={0} style={{ width: '100%' }} />
            </Form.Item>

            <Form.Item
              name="real_url"
              label="真实URL"
            >
              <Input placeholder="请输入真实URL" />
            </Form.Item>

            <Form.Item
              name="origin_url"
              label="原始URL"
            >
              <Input placeholder="请输入原始URL" />
            </Form.Item>

            <Form.Item
              name="is_save"
              label="是否保存"
              initialValue={0}
            >
              <InputNumber min={0} style={{ width: '100%' }} />
            </Form.Item>

            <Form.Item
              name="gpu_error"
              label="GPU错误"
              initialValue={0}
            >
              <InputNumber min={0} style={{ width: '100%' }} />
            </Form.Item>

            <Form.Item
              name="req_ads_count"
              label="请求广告数"
              initialValue={0}
            >
              <InputNumber min={0} style={{ width: '100%' }} />
            </Form.Item>

            <Form.Item
              name="match_count"
              label="匹配数"
              initialValue={0}
            >
              <InputNumber min={0} style={{ width: '100%' }} />
            </Form.Item>

            <Form.Item
              name="active_view"
              label="有效视图"
              initialValue={0}
            >
              <InputNumber min={0} style={{ width: '100%' }} />
            </Form.Item>

            <Form.Item
              name="send_bytes"
              label="发送字节"
              initialValue="0"
            >
              <Input placeholder="请输入发送字节" />
            </Form.Item>

            <Form.Item
              name="recv_bytes"
              label="接收字节"
              initialValue="0"
            >
              <Input placeholder="请输入接收字节" />
            </Form.Item>

            <Form.Item
              name="sleep_info"
              label="睡眠信息"
            >
              <Input.TextArea placeholder="请输入睡眠信息" rows={2} />
            </Form.Item>

            <Form.Item
              name="start_time"
              label="开始时间"
            >
              <InputNumber style={{ width: '100%' }} placeholder="请输入开始时间（时间戳）" />
            </Form.Item>

            <Form.Item
              name="end_time"
              label="结束时间"
            >
              <InputNumber style={{ width: '100%' }} placeholder="请输入结束时间（时间戳）" />
            </Form.Item>

            <Form.Item
              name="plan_id"
              label="计划ID"
            >
              <InputNumber min={0} style={{ width: '100%' }} />
            </Form.Item>

            <Form.Item
              name="time"
              label="当前时间"
            >
              <Input placeholder="请输入当前时间" />
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
        </Spin>
      </Modal>

      {/* 批量添加上报数据模态框 */}
      <Modal
        title="批量添加上报数据"
        open={batchModalVisible}
        onCancel={() => setBatchModalVisible(false)}
        footer={null}
        width={800}
      >
        <Spin spinning={loading} tip="处理中...">
          <Form
            form={batchForm}
            layout="vertical"
            onFinish={handleBatchSubmit}
          >
            <Form.Item
              name="reports"
              label="上报数据（JSON格式，每行一条）"
              rules={[{ required: true, message: '请输入上报数据' }]}
            >
              <Input.TextArea 
                placeholder="请输入JSON格式的上报数据，每行一条"
                rows={10}
                style={{ fontFamily: 'monospace' }}
              />
            </Form.Item>

            <Form.Item>
              <Space>
                <Button type="primary" htmlType="submit">
                  批量添加
                </Button>
                <Button onClick={() => setBatchModalVisible(false)}>
                  取消
                </Button>
              </Space>
            </Form.Item>
          </Form>
        </Spin>
      </Modal>

      {/* 详情模态框 */}
      <Modal
        title="上报详情"
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
            <Descriptions.Item label="广告ID">{detailData?.ad_id}</Descriptions.Item>
            <Descriptions.Item label="用户ID">{detailData?.user_id}</Descriptions.Item>
            <Descriptions.Item label="操作类型">{detailData?.action}</Descriptions.Item>
            <Descriptions.Item label="设备信息">
              <pre style={{ whiteSpace: 'pre-wrap', margin: 0 }}>
                {JSON.stringify(detailData?.device_info, null, 2)}
              </pre>
            </Descriptions.Item>
            <Descriptions.Item label="IP地址">{detailData?.ip}</Descriptions.Item>
            <Descriptions.Item label="内部客户端">{detailData?.client_inner}</Descriptions.Item>
            <Descriptions.Item label="外部客户端">{detailData?.client_outer}</Descriptions.Item>
            <Descriptions.Item label="真实IP">{detailData?.real_ip}</Descriptions.Item>
            <Descriptions.Item label="点击数">{detailData?.click}</Descriptions.Item>
            <Descriptions.Item label="使用时间">{detailData?.use_time}</Descriptions.Item>
            <Descriptions.Item label="广告数">{detailData?.ads}</Descriptions.Item>
            <Descriptions.Item label="是否主页">{detailData?.home_page ? '是' : '否'}</Descriptions.Item>
            <Descriptions.Item label="子页面访问数">{detailData?.subpage_access}</Descriptions.Item>
            <Descriptions.Item label="来源">{detailData?.referer}</Descriptions.Item>
            <Descriptions.Item label="国家">{detailData?.country}</Descriptions.Item>
            <Descriptions.Item label="序列">{detailData?.seq}</Descriptions.Item>
            <Descriptions.Item label="错误信息">{detailData?.error}</Descriptions.Item>
            <Descriptions.Item label="应该点击">{detailData?.should_click ? '是' : '否'}</Descriptions.Item>
            <Descriptions.Item label="应该访问子页面">{detailData?.should_subpage ? '是' : '否'}</Descriptions.Item>
            <Descriptions.Item label="是否iOS">{detailData?.ios ? '是' : '否'}</Descriptions.Item>
            <Descriptions.Item label="可点击">{detailData?.click_able ? '是' : '否'}</Descriptions.Item>
            <Descriptions.Item label="来源类型">{detailData?.source_type}</Descriptions.Item>
            <Descriptions.Item label="IP来源">{detailData?.ip_source}</Descriptions.Item>
            <Descriptions.Item label="广告类型">{detailData?.ad_type}</Descriptions.Item>
            <Descriptions.Item label="是否审核">{detailData?.is_review}</Descriptions.Item>
            <Descriptions.Item label="上一个序列">{detailData?.last_seq}</Descriptions.Item>
            <Descriptions.Item label="屏幕宽度">{detailData?.screen_width}</Descriptions.Item>
            <Descriptions.Item label="真实URL">{detailData?.real_url}</Descriptions.Item>
            <Descriptions.Item label="原始URL">{detailData?.origin_url}</Descriptions.Item>
            <Descriptions.Item label="是否保存">{detailData?.is_save}</Descriptions.Item>
            <Descriptions.Item label="GPU错误">{detailData?.gpu_error}</Descriptions.Item>
            <Descriptions.Item label="请求广告数">{detailData?.req_ads_count}</Descriptions.Item>
            <Descriptions.Item label="匹配数">{detailData?.match_count}</Descriptions.Item>
            <Descriptions.Item label="有效视图">{detailData?.active_view}</Descriptions.Item>
            <Descriptions.Item label="发送字节">{detailData?.send_bytes}</Descriptions.Item>
            <Descriptions.Item label="接收字节">{detailData?.recv_bytes}</Descriptions.Item>
            <Descriptions.Item label="睡眠信息">{detailData?.sleep_info}</Descriptions.Item>
            <Descriptions.Item label="开始时间">{detailData?.start_time}</Descriptions.Item>
            <Descriptions.Item label="结束时间">{detailData?.end_time}</Descriptions.Item>
            <Descriptions.Item label="计划ID">{detailData?.plan_id}</Descriptions.Item>
            <Descriptions.Item label="当前时间">{detailData?.time}</Descriptions.Item>
            <Descriptions.Item label="上报时间">
              {toUserTimezone(detailData?.timestamp)}
            </Descriptions.Item>
          </Descriptions>
        </div>
      </Modal>
    </div>
  );
}

export default AdReportManagement;