package response

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

var builtinZhCNMessages = map[string]string{
	"Success":                    "成功",
	"success":                    "成功",
	"ok":                         "正常",
	"Welcome to AD Platform API": "欢迎使用广告平台 API",
	"Service is running":         "服务运行正常",
	"Login successful":           "登录成功",
	"Registration submitted, pending admin review":           "注册申请已提交，等待管理员审核",
	"Registration already exists":                            "注册申请已存在",
	"Report submitted successfully":                          "上报成功",
	"Task report submitted successfully":                     "任务上报成功",
	"Batch reports submitted successfully":                   "批量上报成功",
	"Batch reports exceed maximum allowed count":           "批量上报条数超过上限",
	"Today reports retrieved successfully":                   "今日上报获取成功",
	"Report stats retrieved successfully":                    "报表统计获取成功",
	"Ad reports retrieved successfully":                      "广告上报获取成功",
	"Ad tasks retrieved successfully":                        "广告任务获取成功",
	"Ad task retrieved successfully":                         "广告任务获取成功",
	"Ad task created successfully":                           "广告任务创建成功",
	"Ad task updated successfully":                           "广告任务更新成功",
	"Ad task deleted successfully":                           "广告任务删除成功",
	"Active ad tasks retrieved successfully":                 "活跃广告任务获取成功",
	"Users retrieved successfully":                           "用户列表获取成功",
	"User retrieved successfully":                            "用户信息获取成功",
	"User created successfully":                              "用户创建成功",
	"User updated successfully":                              "用户更新成功",
	"User deleted successfully":                              "用户删除成功",
	"User enabled successfully":                              "用户启用成功",
	"User disabled successfully":                             "用户禁用成功",
	"User status logs retrieved successfully":                "用户状态日志获取成功",
	"Clients retrieved successfully":                         "客户列表获取成功",
	"Client retrieved successfully":                          "客户信息获取成功",
	"Client created successfully":                            "客户创建成功",
	"Client updated successfully":                            "客户更新成功",
	"Client deleted successfully":                            "客户删除成功",
	"Departments retrieved successfully":                     "部门列表获取成功",
	"Department retrieved successfully":                      "部门信息获取成功",
	"Department created successfully":                        "部门创建成功",
	"Department updated successfully":                        "部门更新成功",
	"Department archived successfully":                       "部门归档成功",
	"Roles retrieved successfully":                           "角色列表获取成功",
	"Role retrieved successfully":                            "角色信息获取成功",
	"Role created successfully":                              "角色创建成功",
	"Role updated successfully":                              "角色更新成功",
	"Role deleted successfully":                              "角色删除成功",
	"Permissions assigned successfully":                      "权限分配成功",
	"Roles assigned successfully":                            "角色分配成功",
	"Permissions retrieved successfully":                     "权限列表获取成功",
	"Unauthorized":                                           "未授权",
	"Invalid operator identity":                              "操作人身份无效",
	"Invalid or expired token":                               "无效或已过期的令牌",
	"Authorization header is required":                       "缺少 Authorization 请求头",
	"Bearer token is required":                               "缺少 Bearer Token",
	"Authorization header must use Bearer token":             "Authorization 必须使用 Bearer Token 格式",
	"X-API-Key header is required":                           "缺少 X-API-Key 请求头",
	"Invalid API key":                                        "无效的 API Key",
	"API key scope denied":                                   "API Key 权限范围不足",
	"Invalid request data":                                   "请求数据不合法",
	"Invalid query parameters":                               "查询参数不合法",
	"Invalid user ID":                                        "用户 ID 不合法",
	"User not found":                                         "用户不存在",
	"Invalid user data":                                      "用户数据不合法",
	"Invalid ad task data":                                   "广告任务数据不合法",
	"Invalid client data":                                    "客户数据不合法",
	"Invalid department data":                                "部门数据不合法",
	"Invalid role data":                                      "角色数据不合法",
	"Failed to retrieve users":                               "获取用户失败",
	"Failed to retrieve user":                                "获取用户失败",
	"Failed to create user":                                  "创建用户失败",
	"Failed to retrieve registrations":                       "获取注册申请失败",
	"Pending registrations retrieved successfully":           "待审核注册申请获取成功",
	"Invalid registration ID":                                "注册申请 ID 不合法",
	"Invalid API key ID":                                     "API Key ID 不合法",
	"API key not found":                                      "API Key 不存在",
	"Invalid API key scope":                                  "API Key 权限范围不合法",
	"Invalid expires_in_days, must be >= 0":                  "expires_in_days 不合法，必须大于等于 0",
	"Invalid expires_in_days, must be <= 3650":               "expires_in_days 不合法，必须小于等于 3650",
	"Review reason is required when rejected":                "驳回时必须填写审核原因",
	"Registration has already been reviewed":                 "该注册申请已审核",
	"Failed to review registration":                          "审核注册申请失败",
	"Registration approval conflict, user may already exist": "审核冲突，用户可能已存在",
	"Registration reviewed successfully":                     "注册申请审核成功",
	"Failed to retrieve registration audits":                 "获取审核记录失败",
	"Registration audits retrieved successfully":             "审核记录获取成功",
	"Failed to retrieve API keys":                            "获取 API Key 失败",
	"API keys retrieved successfully":                        "API Key 列表获取成功",
	"Failed to rotate API key":                               "轮换 API Key 失败",
	"API key rotated successfully":                           "API Key 轮换成功",
	"Failed to disable API key":                              "禁用 API Key 失败",
	"API key disabled successfully":                          "API Key 禁用成功",
	"Failed to update user":                                  "更新用户失败",
	"Failed to delete user":                                  "删除用户失败",
	"Failed to enable user":                                  "启用用户失败",
	"Failed to disable user":                                 "禁用用户失败",
	"Failed to retrieve clients":                             "获取客户失败",
	"Failed to retrieve client":                              "获取客户失败",
	"Failed to retrieve departments":                         "获取部门失败",
	"Failed to retrieve department":                          "获取部门失败",
	"Failed to retrieve roles":                               "获取角色失败",
	"Failed to retrieve role":                                "获取角色失败",
	"Failed to retrieve ad tasks":                            "获取广告任务失败",
	"Failed to retrieve ad task":                             "获取广告任务失败",
	"Failed to create ad task":                               "创建广告任务失败",
	"Failed to update ad task":                               "更新广告任务失败",
	"Failed to delete ad task":                               "删除广告任务失败",
	"Failed to retrieve user status logs":                    "获取用户状态日志失败",
	"Failed to create client":                                "创建客户失败",
	"Failed to update client":                                "更新客户失败",
	"Failed to delete client":                                "删除客户失败",
	"Failed to create department":                            "创建部门失败",
	"Failed to update department":                            "更新部门失败",
	"Failed to archive department":                           "归档部门失败",
	"Failed to create role":                                  "创建角色失败",
	"Failed to update role":                                  "更新角色失败",
	"Failed to delete role":                                  "删除角色失败",
	"Failed to assign permissions":                           "分配权限失败",
	"Failed to assign roles":                                 "分配角色失败",
	"Failed to retrieve permissions":                         "获取权限失败",
	"Failed to retrieve today's reports":                     "获取今日上报失败",
	"Failed to retrieve report stats":                        "获取报表统计失败",
	"Failed to retrieve ad reports":                          "获取广告上报失败",
	"Failed to submit batch reports":                         "批量上报失败",
	"Failed to submit task report":                           "任务上报失败",
	"Invalid username or password":                           "用户名或密码错误",
	"User is disabled":                                       "用户已禁用",
	"Failed to generate token":                               "生成令牌失败",
	"Invalid ad ID":                                          "广告 ID 不合法",
	"Ad task not found":                                      "广告任务不存在",
	"Client not found":                                       "客户不存在",
	"Role not found":                                         "角色不存在",
	"Department not found":                                   "部门不存在",
	"Ad name is required":                                    "广告名称不能为空",
	"Start time is required":                                 "开始时间不能为空",
	"End time is required":                                   "结束时间不能为空",
	"End time must be after start time":                      "结束时间必须晚于开始时间",
	"Failed to serialize data":                               "数据序列化失败",
	"Failed to encrypt data":                                 "数据加密失败",
	"Failed to decrypt data":                                 "数据解密失败",
	"Failed to parse decrypted data":                         "解密数据解析失败",
	"Invalid report data":                                    "上报数据不合法",
	"Failed to submit report":                                "上报失败",
	"Empty data field":                                       "data 字段为空",
	"Failed to retrieve ads":                                 "获取广告失败",
	"Ad not found":                                           "广告不存在",
	"User not authenticated":                                 "用户未认证",
	"Invalid user principal in session":                      "会话用户主体无效",
	"Insufficient permissions":                               "权限不足",
	"User is not assigned to any department":                 "用户未分配部门",
	"Public registration is disabled":                        "公开注册已关闭",
	"Password does not meet minimum requirements":            "密码不符合长度或复杂度要求",
}

var builtinEnUSMessages = map[string]string{
	"已经锁定 不能登录。请联系管理员": "User is locked and cannot log in. Please contact administrator.",
	"请求过于频繁，请稍后再试":     "Too many requests, please try again later.",
	"服务暂时不可用，请稍后再试":    "Service temporarily unavailable, please try again later.",
}

var (
	i18nOnce       sync.Once
	zhCNMessages   map[string]string
	enUSMessages   map[string]string
	missingI18NLog sync.Map
)

func initI18N() {
	zhCNMessages = cloneMessages(builtinZhCNMessages)
	enUSMessages = cloneMessages(builtinEnUSMessages)

	mergeExternalMessages("zh-CN", zhCNMessages)
	mergeExternalMessages("en-US", enUSMessages)
}

func cloneMessages(src map[string]string) map[string]string {
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func mergeExternalMessages(lang string, target map[string]string) {
	path := filepath.Join("config", "i18n", lang+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var messages map[string]string
	if err := json.Unmarshal(data, &messages); err != nil {
		return
	}
	for k, v := range messages {
		target[k] = v
	}
}

func localizeMessage(c *gin.Context, message string) string {
	i18nOnce.Do(initI18N)
	if message == "" {
		return message
	}
	lang := resolveLang(c)
	if lang == "en-US" {
		if v, ok := enUSMessages[message]; ok {
			return v
		}
		reportMissingTranslation("en-US", message)
		return message
	}
	if v, ok := zhCNMessages[message]; ok {
		return v
	}
	reportMissingTranslation("zh-CN", message)
	return message
}

// Translate 对外暴露的消息翻译函数，供非 response 包调用。
func Translate(c *gin.Context, message string) string {
	return localizeMessage(c, message)
}

func resolveLang(c *gin.Context) string {
	if c == nil {
		return "zh-CN"
	}
	if v, ok := c.Get("lang"); ok {
		if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
			if strings.HasPrefix(strings.ToLower(strings.TrimSpace(s)), "en") {
				return "en-US"
			}
		}
	}
	return "zh-CN"
}

func reportMissingTranslation(lang, message string) {
	key := lang + "::" + message
	if _, loaded := missingI18NLog.LoadOrStore(key, struct{}{}); loaded {
		return
	}
	log.Printf("i18n missing translation: lang=%s message=%q", lang, message)
}

// ListMissingTranslations 返回当前进程记录到的缺失翻译，按语言分组。
func ListMissingTranslations() map[string][]string {
	result := map[string][]string{
		"zh-CN": {},
		"en-US": {},
	}
	missingI18NLog.Range(func(key, _ interface{}) bool {
		k, ok := key.(string)
		if !ok {
			return true
		}
		parts := strings.SplitN(k, "::", 2)
		if len(parts) != 2 {
			return true
		}
		lang := parts[0]
		msg := parts[1]
		result[lang] = append(result[lang], msg)
		return true
	})
	for lang := range result {
		sort.Strings(result[lang])
	}
	return result
}

// ResetMissingTranslations 清空当前进程记录的缺失翻译。
func ResetMissingTranslations() {
	missingI18NLog.Range(func(key, _ interface{}) bool {
		missingI18NLog.Delete(key)
		return true
	})
}
