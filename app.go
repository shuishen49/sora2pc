package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// 应用版本号 - 每次发布新版本时更新此值
const AppVersion = "0.0.6"

// GitHub 仓库信息
const (
	GitHubOwner = "shuishen49"
	GitHubRepo  = "sora2pc"
)

// App struct
type App struct {
	ctx context.Context
	db  *sql.DB
	fileServerOnce sync.Once
	fileServerPort int
}

func isProPlan(planType string) bool {
	planType = strings.ToLower(strings.TrimSpace(planType))
	if planType == "" {
		return false
	}
	return strings.Contains(planType, "pro")
}

// Config 用于当 SQLite 不可用时的文件配置回退
type Config struct {
	BaseURL string `json:"base_url"`
}

// HealthResult 用于前端测试服务器健康状态的返回结构
type HealthResult struct {
	Ok      bool   `json:"ok"`
	Message string `json:"message"`
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	// 尝试初始化 SQLite，失败时自动降级为文件存储
	if err := a.initDB(); err != nil {
		runtime.LogWarning(a.ctx, fmt.Sprintf("初始化数据库失败，将使用文件配置: %v", err))
	}
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

// LogDebug 供前端打印调试信息，从命令行运行 exe 时会在终端看到
func (a *App) LogDebug(msg string) {
	runtime.LogInfo(a.ctx, "[前端] "+msg)
}

// initDB 初始化 SQLite 数据库和表
func (a *App) initDB() error {
	// 所有本地数据统一写入当前工作目录下的 accounts.db
	baseDir, err := os.Getwd()
	if err != nil {
		baseDir = "."
	}
	dbPath := filepath.Join(baseDir, "accounts.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		// 在 CGO_DISABLED 环境下，go-sqlite3 会返回 stub 错误，这里直接忽略并退回文件存储
		if strings.Contains(err.Error(), "requires cgo") {
			runtime.LogWarning(a.ctx, "CGO 被禁用，SQLite 将不可用，使用文件配置代替")
			return nil
		}
		return err
	}

	// 创建表（如不存在）
	schema := `
CREATE TABLE IF NOT EXISTS accounts (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	bearer_token TEXT NOT NULL,
	host TEXT,
	port INTEGER,
	status_json TEXT,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS settings (
	key TEXT PRIMARY KEY,
	value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS tokens (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	token TEXT NOT NULL,
	st TEXT,
	rt TEXT,
	client_id TEXT,
	is_active INTEGER DEFAULT 1,
	remark TEXT,
	proxy_url TEXT,
	image_enabled INTEGER DEFAULT 1,
	video_enabled INTEGER DEFAULT 1,
	image_concurrency INTEGER DEFAULT -1,
	video_concurrency INTEGER DEFAULT 3,
	status_json TEXT,
	plan_type TEXT,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS video_task_results (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	task_id TEXT UNIQUE NOT NULL,
	token_id INTEGER NOT NULL,
	result_json TEXT,
	progress_pct REAL DEFAULT 0,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS video_downloads (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	generation_id TEXT UNIQUE NOT NULL,
	task_id TEXT,
	post_id TEXT,
	downloadable_url TEXT,
	local_path TEXT NOT NULL,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS task_list (
	key TEXT PRIMARY KEY,
	value TEXT NOT NULL
);
`
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		// 同样处理可能的 stub 错误
		if strings.Contains(err.Error(), "requires cgo") {
			runtime.LogWarning(a.ctx, "CGO 被禁用，SQLite 将不可用，使用文件配置代替")
			return nil
		}
		return err
	}
	// 兼容旧库：若无 plan_type 列则添加（忽略已存在错误）
	_, _ = db.Exec("ALTER TABLE tokens ADD COLUMN plan_type TEXT DEFAULT ''")
	// 兼容旧库：video_task_results 若无 progress_pct 则添加
	_, _ = db.Exec("ALTER TABLE video_task_results ADD COLUMN progress_pct REAL DEFAULT 0")
	// 兼容旧库：video_task_results 若无 prompt 列则添加
	_, _ = db.Exec("ALTER TABLE video_task_results ADD COLUMN prompt TEXT DEFAULT ''")
	// 兼容旧库：video_downloads 若无 post_id 列则添加
	_, _ = db.Exec("ALTER TABLE video_downloads ADD COLUMN post_id TEXT DEFAULT ''")

	a.db = db
	return nil
}

// loadConfig 从本地 config.json 读取配置（回退方案）
func (a *App) loadConfig() (*Config, error) {
	baseDir, err := os.Getwd()
	if err != nil {
		baseDir = "."
	}
	path := filepath.Join(baseDir, "config.json")

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Config{}, nil
	}
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return &cfg, err
	}
	return &cfg, nil
}

// saveConfig 将配置写入本地 config.json（回退方案）
func (a *App) saveConfig(cfg *Config) error {
	baseDir, err := os.Getwd()
	if err != nil {
		baseDir = "."
	}
	path := filepath.Join(baseDir, "config.json")

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// GetBaseURL 优先从 SQLite settings 表读取 BaseURL，若不可用则回退到 config.json，最后使用默认值
func (a *App) GetBaseURL() string {
	const defaultURL = "http://127.0.0.1:8000"

	// 1) 优先从 SQLite 读取（若可用）
	if a.db != nil {
		var val string
		err := a.db.QueryRow(`SELECT value FROM settings WHERE key = 'base_url'`).Scan(&val)
		if err == nil {
			val = strings.TrimSpace(val)
			if val != "" {
				return val
			}
		} else if err != sql.ErrNoRows {
			runtime.LogError(a.ctx, fmt.Sprintf("读取 base_url 失败: %v", err))
		}
	}

	// 2) 回退到本地 config.json
	if cfg, err := a.loadConfig(); err == nil && strings.TrimSpace(cfg.BaseURL) != "" {
		return strings.TrimSpace(cfg.BaseURL)
	} else if err != nil {
		runtime.LogError(a.ctx, fmt.Sprintf("读取文件配置失败: %v", err))
	}

	// 3) 最终使用默认值
	return defaultURL
}

func (a *App) getSettingValue(key string) string {
	if a.db == nil {
		return ""
	}
	var val string
	if err := a.db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&val); err == nil {
		return val
	}
	return ""
}

func (a *App) setSettingValue(key, value string) {
	if a.db == nil {
		return
	}
	_, _ = a.db.Exec(`INSERT INTO settings (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value=excluded.value`, key, value)
}

// SetBaseURL 将 BaseURL 写入 SQLite settings 表，若 SQLite 不可用则写入 config.json
func (a *App) SetBaseURL(url string) error {
	trimmed := strings.TrimSpace(url)

	// 1) 若 SQLite 可用，先写入 settings 表
	if a.db != nil {
		_, err := a.db.Exec(
			`INSERT INTO settings (key, value) VALUES ('base_url', ?) 
			 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
			trimmed,
		)
		if err != nil {
			runtime.LogError(a.ctx, fmt.Sprintf("保存 base_url 到 SQLite 失败: %v", err))
		}
	}

	// 2) 无论 SQLite 是否成功，都写一份到 config.json 作为通用回退
	cfg, err := a.loadConfig()
	if err != nil {
		// 如果连读都失败，就直接覆盖写新配置
		cfg = &Config{}
	}
	cfg.BaseURL = trimmed
	if err := a.saveConfig(cfg); err != nil {
		runtime.LogError(a.ctx, fmt.Sprintf("保存 base_url 到文件失败: %v", err))
		return err
	}

	return nil
}

// CheckAccountAndSave 调用 /account/status 并将账号信息写入本地 SQLite
func (a *App) CheckAccountAndSave(bearerToken string) (string, error) {
	if bearerToken == "" {
		return "", fmt.Errorf("bearer_token 不能为空")
	}

	base := strings.TrimRight(a.GetBaseURL(), "/")
	statusURL := base + "/account/status"

	payload := map[string]string{
		"bearer_token": bearerToken,
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest(http.MethodPost, statusURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	runtime.LogInfo(a.ctx, fmt.Sprintf("请求账号状态: %s", statusURL))

	resp, err := client.Do(req)
	if err != nil {
		runtime.LogError(a.ctx, fmt.Sprintf("请求账号状态失败: %v", err))
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		runtime.LogError(a.ctx, fmt.Sprintf("读取账号状态响应失败: %v", err))
		return "", err
	}

	runtime.LogInfo(a.ctx, fmt.Sprintf("账号状态响应: HTTP %d, Body: %s", resp.StatusCode, string(respBody)))

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("账号状态检查失败: HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	// 解析 host/port 方便后续查询
	u, err := url.Parse(base)
	var host string
	var port int
	if err == nil {
		host = u.Hostname()
		if p := u.Port(); p != "" {
			fmt.Sscanf(p, "%d", &port)
		}
	}

	// 将结果写入 SQLite
	if a.db != nil {
		_, err = a.db.Exec(
			`INSERT INTO accounts (bearer_token, host, port, status_json, created_at) VALUES (?, ?, ?, ?, ?)`,
			bearerToken,
			host,
			port,
			string(respBody),
			time.Now(),
		)
		if err != nil {
			runtime.LogError(a.ctx, fmt.Sprintf("写入账号到 SQLite 失败: %v", err))
		}
	} else {
		runtime.LogError(a.ctx, "SQLite 数据库未初始化，无法写入账号信息")
	}

	return string(respBody), nil
}

// AccountMe 调用远程服务器的 POST /account/me，使用 GetBaseURL()（填好的远程地址）
// 请求体: {"bearer_token": "..."}，返回 profile / my_info（含 email 等）
func (a *App) AccountMe(bearerToken string) (string, error) {
	if bearerToken == "" {
		return "", fmt.Errorf("bearer_token 不能为空")
	}
	base := strings.TrimRight(a.GetBaseURL(), "/")
	meURL := base + "/account/me"
	payload := map[string]string{"bearer_token": bearerToken}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest(http.MethodPost, meURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	runtime.LogInfo(a.ctx, fmt.Sprintf("请求账号资料: %s", meURL))
	resp, err := client.Do(req)
	if err != nil {
		runtime.LogError(a.ctx, fmt.Sprintf("请求 /account/me 失败: %v", err))
		return "", err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	runtime.LogInfo(a.ctx, fmt.Sprintf("/account/me 响应: HTTP %d", resp.StatusCode))
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("account/me 失败, HTTP %d: %s", resp.StatusCode, string(respBody))
	}
	return string(respBody), nil
}

// AccountSubscriptions 调用远程 POST /account/subscriptions，传 bearer_token，返回 data[].plan.id 用于判断 free/plus
func (a *App) AccountSubscriptions(bearerToken string) (string, error) {
	if bearerToken == "" {
		return "", fmt.Errorf("bearer_token 不能为空")
	}
	base := strings.TrimRight(a.GetBaseURL(), "/")
	url := base + "/account/subscriptions"
	payload := map[string]string{"bearer_token": bearerToken}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	runtime.LogInfo(a.ctx, fmt.Sprintf("请求订阅: %s", url))
	resp, err := client.Do(req)
	if err != nil {
		runtime.LogError(a.ctx, fmt.Sprintf("请求 /account/subscriptions 失败: %v", err))
		return "", err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	runtime.LogInfo(a.ctx, fmt.Sprintf("/account/subscriptions 响应: HTTP %d", resp.StatusCode))
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("account/subscriptions 失败, HTTP %d: %s", resp.StatusCode, string(respBody))
	}
	return string(respBody), nil
}

// parsePlanTypeFromSubscriptions 从 /account/subscriptions 响应中取最高 rank 的 plan.id（chatgpt_free/chatgpt_plus 等）
func parsePlanTypeFromSubscriptions(body string) string {
	var res struct {
		Data []struct {
			Plan struct {
				ID   string `json:"id"`
				Rank int    `json:"rank"`
			} `json:"plan"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &res); err != nil || len(res.Data) == 0 {
		return ""
	}
	best := ""
	bestRank := -1
	for _, d := range res.Data {
		if d.Plan.ID != "" && d.Plan.Rank > bestRank {
			bestRank = d.Plan.Rank
			best = d.Plan.ID
		}
	}
	if best != "" {
		return best
	}
	if len(res.Data) > 0 && res.Data[0].Plan.ID != "" {
		return res.Data[0].Plan.ID
	}
	return ""
}

// isLocalApiPath 判断是否为本地管理接口（不应转发到远程 Sora 服务器）
func isLocalApiPath(path string) bool {
	if strings.HasPrefix(path, "/api/tokens") {
		return true
	}
	if strings.HasPrefix(path, "/api/admin") {
		return true
	}
	if strings.HasPrefix(path, "/api/logs") {
		return true
	}
	if strings.HasPrefix(path, "/api/stats") {
		return true
	}
	if strings.HasPrefix(path, "/api/proxy") {
		return true
	}
	if strings.HasPrefix(path, "/api/watermark-free") {
		return true
	}
	if strings.HasPrefix(path, "/api/cache") {
		return true
	}
	if strings.HasPrefix(path, "/api/generation") {
		return true
	}
	if strings.HasPrefix(path, "/api/token-refresh") {
		return true
	}
	if strings.HasPrefix(path, "/api/tasks/") {
		return true
	}
	return false
}

// logApiLabel 根据 path 返回用于控制台打印的标签（create / pending / drafts），空串表示不单独打印
func logApiLabel(path string) string {
	switch {
	case strings.Contains(path, "/videos") || strings.Contains(path, "videos"):
		return "CREATE"
	case strings.Contains(path, "pending"):
		return "PENDING"
	case strings.Contains(path, "drafts"):
		return "DRAFTS"
	default:
		return ""
	}
}

// ApiRequest 通用 API 代理：本地管理接口在 Go 内处理，其余转发到远程服务器
func (a *App) ApiRequest(method string, path string, body string, token string) (string, error) {
	if isLocalApiPath(path) {
		runtime.LogInfo(a.ctx, fmt.Sprintf("ApiRequest 本地处理: %s %s", method, path))
		return a.handleLocalApi(method, path, body, token)
	}

	base := strings.TrimRight(a.GetBaseURL(), "/")
	fullURL := base + path

	label := logApiLabel(path)
	if label != "" {
		runtime.LogInfo(a.ctx, "========== "+label+" 请求 ==========")
		runtime.LogInfo(a.ctx, fmt.Sprintf("  %s %s", method, fullURL))
		if body != "" {
			if len(body) > 2000 {
				runtime.LogInfo(a.ctx, "  body: "+body[:2000]+"...(truncated)")
			} else {
				runtime.LogInfo(a.ctx, "  body: "+body)
			}
		}
		runtime.LogInfo(a.ctx, "----------------------------------------")
	} else {
		runtime.LogInfo(a.ctx, fmt.Sprintf("ApiRequest 转发远程: %s %s", method, fullURL))
	}

	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}

	req, err := http.NewRequest(method, fullURL, bodyReader)
	if err != nil {
		runtime.LogError(a.ctx, fmt.Sprintf("ApiRequest 创建请求失败: %v", err))
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		runtime.LogError(a.ctx, fmt.Sprintf("ApiRequest 请求失败: %v", err))
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		runtime.LogError(a.ctx, fmt.Sprintf("ApiRequest 读取响应失败: %v", err))
		return "", err
	}

	if label != "" {
		runtime.LogInfo(a.ctx, "========== "+label+" 响应 (HTTP "+strconv.Itoa(resp.StatusCode)+") ==========")
		respStr := string(respBody)
		if len(respStr) > 3000 {
			runtime.LogInfo(a.ctx, respStr[:3000]+"...(truncated)")
		} else {
			runtime.LogInfo(a.ctx, respStr)
		}
		runtime.LogInfo(a.ctx, "========================================")
	} else {
		runtime.LogInfo(a.ctx, fmt.Sprintf("ApiRequest 响应: HTTP %d, Body: %s", resp.StatusCode, string(respBody)))
	}

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return string(respBody), nil
}

// handleLocalApi 处理本地管理接口，不转发到远程
func (a *App) handleLocalApi(method string, path string, body string, _ string) (string, error) {
	fullPath := path // 保留完整路径供 list 解析 query
	path = strings.TrimPrefix(path, "/api")
	if idx := strings.Index(path, "?"); idx >= 0 {
		path = path[:idx]
	}
	path = strings.Trim(path, "/")
	parts := strings.Split(path, "/")
	if path == "" {
		parts = nil
	}

	// /api/tokens 系列
	if len(parts) >= 1 && parts[0] == "tokens" {
		return a.handleLocalTokens(method, fullPath, parts, body)
	}
	// /api/admin/config 等
	if len(parts) >= 1 && parts[0] == "admin" {
		return a.handleLocalAdmin(method, path, parts, body)
	}
	// /api/logs
	if len(parts) >= 1 && parts[0] == "logs" {
		return a.handleLocalLogs(method, path, body)
	}
	// /api/stats
	if len(parts) >= 1 && parts[0] == "stats" {
		return a.handleLocalStats(method, path)
	}
	// /api/proxy, /api/watermark-free, /api/cache, /api/generation, /api/token-refresh
	if len(parts) >= 1 {
		switch parts[0] {
		case "proxy", "watermark-free", "cache", "generation", "token-refresh":
			return a.handleLocalConfigDefault(parts[0], method, path, body)
		}
	}
	// /api/tasks/:id/cancel
	if len(parts) >= 1 && parts[0] == "tasks" {
		return a.handleLocalTasks(method, path, parts, body)
	}

	return "", fmt.Errorf("本地接口未实现: %s %s", method, path)
}

// handleLocalTokens 处理 /api/tokens 的本地 CRUD
func (a *App) handleLocalTokens(method string, rawPath string, parts []string, body string) (string, error) {
	if a.db == nil {
		return jsonFail("SQLite 未初始化，无法使用 Token 管理")
	}

	// GET /api/tokens?page=1&limit=20
	if method == http.MethodGet && len(parts) == 1 {
		return a.localTokensList(rawPath)
	}
	// POST /api/tokens
	if method == http.MethodPost && len(parts) == 1 {
		return a.localTokenCreate(body)
	}
	// POST /api/tokens/import
	if method == http.MethodPost && len(parts) == 2 && parts[1] == "import" {
		return a.localTokensImport(body)
	}
	// POST /api/tokens/st2at
	if method == http.MethodPost && len(parts) == 2 && parts[1] == "st2at" {
		return jsonMarshal(map[string]interface{}{"success": false, "message": "本地存储模式下不支持 ST→AT 转换，请直接使用 Access Token"})
	}
	// POST /api/tokens/rt2at
	if method == http.MethodPost && len(parts) == 2 && parts[1] == "rt2at" {
		return jsonMarshal(map[string]interface{}{"success": false, "message": "本地存储模式下不支持 RT→AT 转换，请直接使用 Access Token"})
	}
	// batch
	if len(parts) >= 2 && parts[1] == "batch" {
		return a.localTokensBatch(method, parts, body)
	}
	// /api/tokens/:id
	if len(parts) >= 2 {
		id, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return jsonFail("无效的 token id")
		}
		if method == http.MethodPost && len(parts) == 2 {
			return a.localTokenUpdate(id, body)
		}
		if method == http.MethodDelete && len(parts) == 2 {
			return a.localTokenDelete(id)
		}
		if method == http.MethodPost && len(parts) == 3 {
			switch parts[2] {
			case "test":
				return a.localTokenTest(id)
			case "enable":
				return a.localTokenSetActive(id, true)
			case "disable":
				return a.localTokenSetActive(id, false)
			}
		}
		if (method == http.MethodPut || method == http.MethodPost) && len(parts) == 3 && parts[2] == "status" {
			return a.localTokenSetStatus(id, body)
		}
	}

	return "", fmt.Errorf("未实现的 tokens 请求: %s %s", method, rawPath)
}

func jsonFail(msg string) (string, error) {
	b, _ := json.Marshal(map[string]interface{}{"success": false, "message": msg})
	return string(b), nil
}

func jsonMarshal(v interface{}) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (a *App) localTokensList(rawPath string) (string, error) {
	u, _ := url.Parse(rawPath)
	q := u.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit < 1 {
		limit = 20
	}
	offset := (page - 1) * limit

	var total int
	if err := a.db.QueryRow(`SELECT COUNT(*) FROM tokens`).Scan(&total); err != nil {
		return jsonFail("查询总数失败: " + err.Error())
	}

	rows, err := a.db.Query(
		`SELECT id, token, st, rt, client_id, is_active, remark, proxy_url, image_enabled, video_enabled, image_concurrency, video_concurrency, status_json, plan_type, created_at FROM tokens ORDER BY id LIMIT ? OFFSET ?`,
		limit, offset)
	if err != nil {
		return jsonFail("查询列表失败: " + err.Error())
	}
	defer rows.Close()

	list := []map[string]interface{}{}
	for rows.Next() {
		var id int64
		var token, st, rt, clientID, remark, proxyURL, statusJSON, planType, createdAt sql.NullString
		var isActive, imageEnabled, videoEnabled int
		var imageConcurrency, videoConcurrency int
		if err := rows.Scan(&id, &token, &st, &rt, &clientID, &isActive, &remark, &proxyURL, &imageEnabled, &videoEnabled, &imageConcurrency, &videoConcurrency, &statusJSON, &planType, &createdAt); err != nil {
			continue
		}
		email := ""
		remainingCount := 0
		resetInSeconds := 0
		if statusJSON.String != "" {
			var status struct {
				Email string `json:"email"`
				Rate  struct {
					AccessResetsInSeconds         int  `json:"access_resets_in_seconds"`
					EstimatedNumVideosRemaining   int  `json:"estimated_num_videos_remaining"`
					EstimatedNumPurchasedRemain   int  `json:"estimated_num_purchased_videos_remaining"`
					CreditRemaining               int  `json:"credit_remaining"`
					RateLimitReached              bool `json:"rate_limit_reached"`
				} `json:"rate_limit_and_credit_balance"`
			}
			_ = json.Unmarshal([]byte(statusJSON.String), &status)
			email = status.Email
			remainingCount = status.Rate.EstimatedNumVideosRemaining
			resetInSeconds = status.Rate.AccessResetsInSeconds
		}
		planTypeStr := ""
		if planType.Valid {
			planTypeStr = planType.String
		}
		item := map[string]interface{}{
			"id":                      id,
			"token":                   token.String,
			"st":                      st.String,
			"rt":                      rt.String,
			"client_id":               clientID.String,
			"is_active":               isActive == 1,
			"remark":                  remark.String,
			"proxy_url":                proxyURL.String,
			"image_enabled":            imageEnabled == 1,
			"video_enabled":            videoEnabled == 1,
			"image_concurrency":       imageConcurrency,
			"video_concurrency":       videoConcurrency,
			"email":                   email,
			"is_expired":              false,
			"plan_type":               planTypeStr,
			"sora2_remaining_count":   remainingCount,
			"sora2_total_count":       0,
			"access_resets_in_seconds": resetInSeconds,
			"image_count":             0,
			"video_count":             0,
			"error_count":             0,
			"expiry_time":             nil,
		}
		list = append(list, item)
	}

	out := map[string]interface{}{"result": list, "total": total}
	return jsonMarshal(out)
}

func (a *App) localTokenCreate(body string) (string, error) {
	var input struct {
		Token            string `json:"token"`
		St               string `json:"st"`
		Rt               string `json:"rt"`
		ClientID         string `json:"client_id"`
		ProxyURL         string `json:"proxy_url"`
		Remark           string `json:"remark"`
		ImageEnabled     bool   `json:"image_enabled"`
		VideoEnabled     bool   `json:"video_enabled"`
		ImageConcurrency int    `json:"image_concurrency"`
		VideoConcurrency int    `json:"video_concurrency"`
		StatusResponse   string `json:"status_response"`
	}
	if err := json.Unmarshal([]byte(body), &input); err != nil {
		return jsonFail("请求体解析失败")
	}
	if strings.TrimSpace(input.Token) == "" {
		return jsonFail("token 不能为空")
	}
	now := time.Now()
	statusJSON := strings.TrimSpace(input.StatusResponse)
	res, err := a.db.Exec(
		`INSERT INTO tokens (token, st, rt, client_id, is_active, remark, proxy_url, image_enabled, video_enabled, image_concurrency, video_concurrency, status_json, created_at, updated_at) VALUES (?, ?, ?, ?, 1, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		strings.TrimSpace(input.Token),
		nullStr(input.St),
		nullStr(input.Rt),
		nullStr(input.ClientID),
		nullStr(input.Remark),
		nullStr(input.ProxyURL),
		boolToInt(input.ImageEnabled),
		boolToInt(input.VideoEnabled),
		input.ImageConcurrency,
		input.VideoConcurrency,
		nullStr(statusJSON),
		now, now,
	)
	if err != nil {
		return jsonFail("写入失败: " + err.Error())
	}
	id, _ := res.LastInsertId()
	// 请求 /account/subscriptions 获取 plan_type（free/plus 等）
	if subBody, err := a.AccountSubscriptions(strings.TrimSpace(input.Token)); err == nil {
		planType := parsePlanTypeFromSubscriptions(subBody)
		if planType != "" {
			_, _ = a.db.Exec(`UPDATE tokens SET plan_type=?, updated_at=? WHERE id=?`, planType, time.Now(), id)
		}
	}
	return jsonMarshal(map[string]interface{}{"success": true, "id": id})
}

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (a *App) localTokenUpdate(id int64, body string) (string, error) {
	var input struct {
		Token             string `json:"token"`
		St                string `json:"st"`
		Rt                string `json:"rt"`
		ClientID          string `json:"client_id"`
		ProxyURL          string `json:"proxy_url"`
		Remark            string `json:"remark"`
		ImageEnabled      bool   `json:"image_enabled"`
		VideoEnabled      bool   `json:"video_enabled"`
		ImageConcurrency  *int   `json:"image_concurrency"`
		VideoConcurrency  *int   `json:"video_concurrency"`
	}
	if err := json.Unmarshal([]byte(body), &input); err != nil {
		return jsonFail("请求体解析失败")
	}
	if strings.TrimSpace(input.Token) == "" {
		return jsonFail("token 不能为空")
	}
	imgConc := -1
	if input.ImageConcurrency != nil {
		imgConc = *input.ImageConcurrency
	}
	vidConc := 3
	if input.VideoConcurrency != nil {
		vidConc = *input.VideoConcurrency
	}
	_, err := a.db.Exec(
		`UPDATE tokens SET token=?, st=?, rt=?, client_id=?, remark=?, proxy_url=?, image_enabled=?, video_enabled=?, image_concurrency=?, video_concurrency=?, updated_at=? WHERE id=?`,
		strings.TrimSpace(input.Token), nullStr(input.St), nullStr(input.Rt), nullStr(input.ClientID), nullStr(input.Remark), nullStr(input.ProxyURL),
		boolToInt(input.ImageEnabled), boolToInt(input.VideoEnabled), imgConc, vidConc, time.Now(), id,
	)
	if err != nil {
		return jsonFail("更新失败: " + err.Error())
	}
	return jsonMarshal(map[string]interface{}{"success": true})
}

func (a *App) localTokenDelete(id int64) (string, error) {
	_, err := a.db.Exec(`DELETE FROM tokens WHERE id=?`, id)
	if err != nil {
		return jsonFail("删除失败: " + err.Error())
	}
	return jsonMarshal(map[string]interface{}{"success": true})
}

func (a *App) localTokenTest(id int64) (string, error) {
	var bearer string
	if err := a.db.QueryRow(`SELECT token FROM tokens WHERE id=?`, id).Scan(&bearer); err != nil {
		return jsonFail("Token 不存在")
	}
	respBody, err := a.CheckAccountAndSave(bearer)
	if err != nil {
		return jsonMarshal(map[string]interface{}{
			"success": false,
			"message": err.Error(),
			"status":  "failed",
		})
	}
	// 更新该记录的 status_json
	var status map[string]interface{}
	_ = json.Unmarshal([]byte(respBody), &status)
	statusJSON, _ := json.Marshal(status)
	_, _ = a.db.Exec(`UPDATE tokens SET status_json=?, updated_at=? WHERE id=?`, string(statusJSON), time.Now(), id)
	email, _ := status["email"].(string)
	// 请求 /account/subscriptions 更新 plan_type（free/plus 等）
	if subBody, err := a.AccountSubscriptions(bearer); err == nil {
		planType := parsePlanTypeFromSubscriptions(subBody)
		if planType != "" {
			_, _ = a.db.Exec(`UPDATE tokens SET plan_type=?, updated_at=? WHERE id=?`, planType, time.Now(), id)
		}
	}
	return jsonMarshal(map[string]interface{}{
		"success": true,
		"status":  "success",
		"email":   email,
	})
}

func (a *App) localTokenSetActive(id int64, active bool) (string, error) {
	v := 0
	if active {
		v = 1
	}
	_, err := a.db.Exec(`UPDATE tokens SET is_active=?, updated_at=? WHERE id=?`, v, time.Now(), id)
	if err != nil {
		return jsonFail("更新失败: " + err.Error())
	}
	return jsonMarshal(map[string]interface{}{"success": true})
}

// GetRandomVideoToken 从数据库随机返回一个可用于视频生成的 token：状态正常、已启用视频、有剩余次数
// requirePro=true 时仅允许 Pro/Plus 账号
// 返回 JSON：{"bearer_token": "xxx", "token_id": 123} 或 {"error": "..."}
func (a *App) GetRandomVideoToken(requirePro bool) (string, error) {
	rows, err := a.db.Query(
		`SELECT id, token, status_json, plan_type FROM tokens WHERE is_active=1 AND video_enabled=1`)
	if err != nil {
		return jsonMarshal(map[string]interface{}{"error": "查询 Token 失败: " + err.Error()})
	}
	defer rows.Close()

	type tokenRow struct {
		id         int64
		token      string
		statusJSON sql.NullString
		planType   sql.NullString
	}
	var candidates []tokenRow
	for rows.Next() {
		var id int64
		var token string
		var statusJSON sql.NullString
		var planType sql.NullString
		if err := rows.Scan(&id, &token, &statusJSON, &planType); err != nil {
			continue
		}
		if strings.TrimSpace(token) == "" {
			continue
		}
		if requirePro && (!planType.Valid || !isProPlan(planType.String)) {
			continue
		}
		remaining := -1
		if statusJSON.Valid && statusJSON.String != "" {
			var status struct {
				Rate struct {
					EstimatedNumVideosRemaining int `json:"estimated_num_videos_remaining"`
				} `json:"rate_limit_and_credit_balance"`
			}
			if json.Unmarshal([]byte(statusJSON.String), &status) == nil {
				remaining = status.Rate.EstimatedNumVideosRemaining
			}
		}
		// 无 status 时也加入候选（由上游判断）；有 status 时要求剩余次数 > 0
		if remaining < 0 || remaining > 0 {
			candidates = append(candidates, tokenRow{id: id, token: token, statusJSON: statusJSON, planType: planType})
		}
	}
	if len(candidates) == 0 {
		if requirePro {
			return jsonMarshal(map[string]interface{}{"error": "无可用 Pro Token（需状态正常、已启用视频且有剩余次数）"})
		}
		return jsonMarshal(map[string]interface{}{"error": "无可用 Token（需状态正常、已启用视频且有剩余次数）"})
	}
	idx := rand.Intn(len(candidates))
	c := candidates[idx]
	return jsonMarshal(map[string]interface{}{"bearer_token": c.token, "token_id": c.id})
}

// GetBearerByTokenID 根据 token_id 查询该账号的 bearer token，供 pending 轮询时使用「创建任务时的同一账号」
// 返回 JSON：{"bearer_token": "xxx"} 或 {"error": "..."}
func (a *App) GetBearerByTokenID(tokenId int64) (string, error) {
	var bearer string
	if err := a.db.QueryRow(`SELECT token FROM tokens WHERE id=?`, tokenId).Scan(&bearer); err != nil {
		return jsonMarshal(map[string]interface{}{"error": "Token 不存在或已删除"})
	}
	if strings.TrimSpace(bearer) == "" {
		return jsonMarshal(map[string]interface{}{"error": "Token 为空"})
	}
	return jsonMarshal(map[string]interface{}{"bearer_token": bearer})
}

// GetTokenEmailByID 根据 token_id 读取该账号邮箱（来自 status_json）
// 返回 JSON：{"email": "xxx"} 或 {"error": "..."}
func (a *App) GetTokenEmailByID(tokenId int64) (string, error) {
	var statusJSON sql.NullString
	if err := a.db.QueryRow(`SELECT status_json FROM tokens WHERE id=?`, tokenId).Scan(&statusJSON); err != nil {
		return jsonMarshal(map[string]interface{}{"error": "Token 不存在或已删除"})
	}
	if !statusJSON.Valid || statusJSON.String == "" {
		return jsonMarshal(map[string]interface{}{"error": "未找到邮箱"})
	}
	var status struct {
		Email string `json:"email"`
	}
	if err := json.Unmarshal([]byte(statusJSON.String), &status); err != nil || status.Email == "" {
		return jsonMarshal(map[string]interface{}{"error": "未找到邮箱"})
	}
	return jsonMarshal(map[string]interface{}{"email": status.Email})
}

// CreateVideo 调用与 testsh/create.sh 相同的接口：POST {apiBaseURL}/videos，请求体为 bearer_token、prompt、orientation、size、n_frames、model
// 用于「立即生成」视频任务，并在控制台打印 CREATE 请求/响应
// orientation: portrait / landscape；nFrames: 300(10s) / 450(15s) / 750(25s)
func (a *App) CreateVideo(apiBaseURL string, bearerToken string, prompt string, orientation string, nFrames string, model string, size string) (string, error) {
	apiBaseURL = strings.TrimRight(apiBaseURL, "/")
	videoURL := apiBaseURL + "/videos"
	nFramesInt := 300
	if n := strings.TrimSpace(nFrames); n != "" {
		if v, err := strconv.Atoi(n); err == nil && v > 0 {
			nFramesInt = v
		}
	}
	if orientation == "" {
		orientation = "portrait"
	}
	if strings.TrimSpace(model) == "" {
		model = "sy_8"
	}
	if strings.TrimSpace(size) == "" {
		size = "small"
	}
	body := map[string]interface{}{
		"bearer_token": bearerToken,
		"prompt":       prompt,
		"orientation": orientation,
		"size":         size,
		"n_frames":     nFramesInt,
		"model":        model,
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	runtime.LogInfo(a.ctx, "========== CREATE 请求 (POST /videos) ==========")
	runtime.LogInfo(a.ctx, "  "+videoURL)
	runtime.LogInfo(a.ctx, "  prompt="+prompt+" orientation="+orientation+" n_frames="+strconv.Itoa(nFramesInt)+" model="+model+" size="+size)
	runtime.LogInfo(a.ctx, "----------------------------------------")

	req, err := http.NewRequest(http.MethodPost, videoURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		runtime.LogError(a.ctx, "CREATE 请求失败: "+err.Error())
		return "", err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	respStr := string(respBody)
	runtime.LogInfo(a.ctx, "========== CREATE 响应 (HTTP "+strconv.Itoa(resp.StatusCode)+") ==========")
	if len(respStr) > 2000 {
		runtime.LogInfo(a.ctx, respStr[:2000]+"...(truncated)")
	} else {
		runtime.LogInfo(a.ctx, respStr)
	}
	runtime.LogInfo(a.ctx, "========================================")

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, respStr)
	}
	return respStr, nil
}

// PollPending 调用与 testsh/test_pending.sh 相同的接口：POST {apiBaseURL}/pending，请求体为 bearer_token
// 返回 pending 列表 JSON；返回 [] 表示任务已完成。用于 CreateVideo 成功后每 10s 轮询一次
func (a *App) PollPending(apiBaseURL string, bearerToken string) (string, error) {
	apiBaseURL = strings.TrimRight(apiBaseURL, "/")
	pendingURL := apiBaseURL + "/pending"
	body := map[string]string{"bearer_token": bearerToken}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	runtime.LogInfo(a.ctx, "========== PENDING 请求 (POST /pending) ==========")
	runtime.LogInfo(a.ctx, "  "+pendingURL)
	runtime.LogInfo(a.ctx, "----------------------------------------")

	req, err := http.NewRequest(http.MethodPost, pendingURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		runtime.LogError(a.ctx, "PENDING 请求失败: "+err.Error())
		return "", err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	respStr := string(respBody)
	runtime.LogInfo(a.ctx, "========== PENDING 响应 (HTTP "+strconv.Itoa(resp.StatusCode)+") ==========")
	if len(respStr) > 2000 {
		runtime.LogInfo(a.ctx, respStr[:2000]+"...(truncated)")
	} else {
		runtime.LogInfo(a.ctx, respStr)
	}
	runtime.LogInfo(a.ctx, "========================================")

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, respStr)
	}
	return respStr, nil
}

// FetchDrafts 调用与 testsh/test_drafts.sh 相同的接口：POST {apiBaseURL}/drafts，请求体为 bearer_token、limit、offset
// 返回 drafts 响应 JSON（含 items），当 pending 返回 [] 后拉取草稿并下载
func (a *App) FetchDrafts(apiBaseURL string, bearerToken string) (string, error) {
	apiBaseURL = strings.TrimRight(apiBaseURL, "/")
	draftsURL := apiBaseURL + "/drafts"
	body := map[string]interface{}{
		"bearer_token": bearerToken,
		"limit":        20,
		"offset":       0,
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	runtime.LogInfo(a.ctx, "========== DRAFTS 请求 (POST /drafts) ==========")
	runtime.LogInfo(a.ctx, "  "+draftsURL)
	runtime.LogInfo(a.ctx, "----------------------------------------")

	req, err := http.NewRequest(http.MethodPost, draftsURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		runtime.LogError(a.ctx, "DRAFTS 请求失败: "+err.Error())
		return "", err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	respStr := string(respBody)
	runtime.LogInfo(a.ctx, "========== DRAFTS 响应 (HTTP "+strconv.Itoa(resp.StatusCode)+") ==========")
	if len(respStr) > 2000 {
		runtime.LogInfo(a.ctx, respStr[:2000]+"...(truncated)")
	} else {
		runtime.LogInfo(a.ctx, respStr)
	}
	runtime.LogInfo(a.ctx, "========================================")

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, respStr)
	}
	return respStr, nil
}

// SaveVideoTaskResult 保存视频创建成功后的结果：写入 video_task_results，并用 estimated_num_videos_remaining、access_resets_in_seconds 更新对应 token 的 status_json
// resultJson 格式示例：{"id":"task_01kg...","rate_limit_and_credit_balance":{"estimated_num_videos_remaining":29,"access_resets_in_seconds":85511,...},...}
func (a *App) SaveVideoTaskResult(tokenId int64, resultJson string, prompt string) (string, error) {
	var result struct {
		ID                      string `json:"id"`
		RateLimitAndCreditBalance *struct {
			EstimatedNumVideosRemaining   int `json:"estimated_num_videos_remaining"`
			AccessResetsInSeconds        int `json:"access_resets_in_seconds"`
			CreditRemaining             int `json:"credit_remaining"`
			RateLimitReached            bool `json:"rate_limit_reached"`
		} `json:"rate_limit_and_credit_balance"`
	}
	if err := json.Unmarshal([]byte(resultJson), &result); err != nil {
		return jsonFail("resultJson 解析失败: " + err.Error())
	}
	taskID := strings.TrimSpace(result.ID)
	if taskID == "" {
		return jsonFail("resultJson 缺少 id (task_id)")
	}
	now := time.Now()
	_, err := a.db.Exec(
		`INSERT OR REPLACE INTO video_task_results (task_id, token_id, result_json, progress_pct, created_at, prompt) VALUES (?, ?, ?, 0, ?, ?)`,
		taskID, tokenId, resultJson, now, strings.TrimSpace(prompt))
	if err != nil {
		return jsonFail("写入 video_task_results 失败: " + err.Error())
	}
	if result.RateLimitAndCreditBalance != nil {
		// 更新该 token 的 status_json：合并 rate_limit 信息（剩余次数、恢复时间）
		rate := result.RateLimitAndCreditBalance
		var status map[string]interface{}
		var statusJSON sql.NullString
		if err := a.db.QueryRow(`SELECT status_json FROM tokens WHERE id=?`, tokenId).Scan(&statusJSON); err == nil && statusJSON.Valid && statusJSON.String != "" {
			_ = json.Unmarshal([]byte(statusJSON.String), &status)
		}
		if status == nil {
			status = make(map[string]interface{})
		}
		rateMap := map[string]interface{}{
			"estimated_num_videos_remaining":    rate.EstimatedNumVideosRemaining,
			"access_resets_in_seconds":           rate.AccessResetsInSeconds,
			"credit_remaining":                   rate.CreditRemaining,
			"rate_limit_reached":                rate.RateLimitReached,
		}
		status["rate_limit_and_credit_balance"] = rateMap
		newJSON, _ := json.Marshal(status)
		_, _ = a.db.Exec(`UPDATE tokens SET status_json=?, updated_at=? WHERE id=?`, string(newJSON), now, tokenId)
	}
	return jsonMarshal(map[string]interface{}{"success": true, "task_id": taskID})
}

// UpdateVideoTaskProgress 更新 video_task_results 中该 task_id 的进度百分比（pending 轮询得到 progress_pct 时调用）
func (a *App) UpdateVideoTaskProgress(taskId string, progressPct float64) (string, error) {
	taskId = strings.TrimSpace(taskId)
	if taskId == "" {
		return jsonFail("task_id 不能为空")
	}
	_, err := a.db.Exec(`UPDATE video_task_results SET progress_pct=? WHERE task_id=?`, progressPct, taskId)
	if err != nil {
		return jsonFail("更新 progress_pct 失败: " + err.Error())
	}
	return jsonMarshal(map[string]interface{}{"success": true})
}

// GetTokenIDByRemoteTaskID 根据 remote_task_id（即 video_task_results.task_id）查询创建该任务时用的 token_id，供页面加载后恢复 pending 时取 bearer
// 返回 JSON：{"token_id": 10} 或 {"error": "..."}
func (a *App) GetTokenIDByRemoteTaskID(remoteTaskId string) (string, error) {
	remoteTaskId = strings.TrimSpace(remoteTaskId)
	if remoteTaskId == "" {
		return jsonMarshal(map[string]interface{}{"error": "remote_task_id 不能为空"})
	}
	var tokenID int64
	if err := a.db.QueryRow(`SELECT token_id FROM video_task_results WHERE task_id=?`, remoteTaskId).Scan(&tokenID); err != nil {
		return jsonMarshal(map[string]interface{}{"error": "未找到该任务记录"})
	}
	return jsonMarshal(map[string]interface{}{"token_id": tokenID})
}

// GetIncompleteVideoTasks 从 SQLite 查询未完成的视频任务（progress_pct < 100），供页面加载时恢复 pending 轮询
// 返回 JSON：{"tasks": [{"task_id": "xxx", "token_id": 10}, ...]}，无数据时 tasks 为空数组；出错时 {"error": "..."}
func (a *App) GetIncompleteVideoTasks() (string, error) {
	if a.db == nil {
		return jsonMarshal(map[string]interface{}{"tasks": []interface{}{}})
	}
	rows, err := a.db.Query(`SELECT task_id, token_id FROM video_task_results WHERE progress_pct < 100 OR progress_pct IS NULL ORDER BY created_at ASC`)
	if err != nil {
		return jsonFail("查询未完成视频任务失败: " + err.Error())
	}
	defer rows.Close()
	var list []map[string]interface{}
	for rows.Next() {
		var taskID string
		var tokenID int64
		if err := rows.Scan(&taskID, &tokenID); err != nil {
			continue
		}
		list = append(list, map[string]interface{}{"task_id": taskID, "token_id": tokenID})
	}
	out := map[string]interface{}{"tasks": list}
	needPending := len(list) > 0
	runtime.LogInfo(a.ctx, fmt.Sprintf("[GetIncompleteVideoTasks] 读取结果: %d 条, 是否需要继续 pending: %v", len(list), needPending))
	outStr, err := jsonMarshal(out)
	if err == nil {
		runtime.LogInfo(a.ctx, "[GetIncompleteVideoTasks] 明细: "+outStr)
	}
	return outStr, err
}

// drafts 响应中的单个 item 结构（仅解析所需字段）
type draftsItem struct {
	ID              string `json:"id"`
	GenerationID    string `json:"generation_id"`
	TaskID         string `json:"task_id"`
	DownloadableURL string `json:"downloadable_url"`
	Prompt         string `json:"prompt"`
}

// SaveDraftsAndDownload 解析 drafts 响应 JSON，仅下载 completedTaskId 对应的那条，写入 video_downloads 表
// completedTaskId 为空则不下任何下载（避免误下全部）；格式：{"items":[{"task_id":"task_01kgg...","downloadable_url":"https://...",...}],"cursor":"..."}
func (a *App) SaveDraftsAndDownload(draftsJson string, completedTaskId string) (string, error) {
	completedTaskId = strings.TrimSpace(completedTaskId)
	if completedTaskId == "" {
		return jsonMarshal(map[string]interface{}{"success": true, "message": "未指定 completedTaskId，跳过下载", "downloaded": 0})
	}
	var drafts struct {
		Items  []draftsItem `json:"items"`
		Cursor string       `json:"cursor"`
	}
	if err := json.Unmarshal([]byte(draftsJson), &drafts); err != nil {
		return jsonFail("draftsJson 解析失败: " + err.Error())
	}
	// 只保留 task_id 与刚完成任务一致的那条
	var target *draftsItem
	for i := range drafts.Items {
		if strings.TrimSpace(drafts.Items[i].TaskID) == completedTaskId {
			target = &drafts.Items[i]
			break
		}
	}
	if target == nil {
		runtime.LogInfo(a.ctx, fmt.Sprintf("[SaveDraftsAndDownload] drafts 中未找到 task_id=%s，跳过下载", completedTaskId))
		return jsonMarshal(map[string]interface{}{"success": true, "message": "drafts 中无对应 task_id", "downloaded": 0})
	}

	baseDir, err := os.Getwd()
	if err != nil {
		baseDir = "."
	}
	downloadDir := filepath.Join(baseDir, "downloads")
	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		return jsonFail("创建下载目录失败: " + err.Error())
	}
	runtime.LogInfo(a.ctx, fmt.Sprintf("[SaveDraftsAndDownload] 下载目录: %s，仅下载 task_id=%s", downloadDir, completedTaskId))

	client := &http.Client{Timeout: 120 * time.Second}
	downloaded := 0
	item := *target
	genID := strings.TrimSpace(item.GenerationID)
	if genID == "" {
		genID = strings.TrimSpace(item.ID)
	}
	if genID != "" {
		urlStr := strings.TrimSpace(item.DownloadableURL)
		if urlStr != "" {
			localPath := filepath.Join(downloadDir, genID+".mp4")
			req, _ := http.NewRequest("GET", urlStr, nil)
			req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
			resp, err := client.Do(req)
			if err == nil && resp.StatusCode == http.StatusOK {
				f, err := os.Create(localPath)
				if err == nil {
					_, err = io.Copy(f, resp.Body)
					resp.Body.Close()
					f.Close()
					if err != nil {
						os.Remove(localPath)
					} else if a.db != nil {
						taskID := strings.TrimSpace(item.TaskID)
						_, _ = a.db.Exec(
							`INSERT OR REPLACE INTO video_downloads (generation_id, task_id, downloadable_url, local_path, created_at) VALUES (?, ?, ?, ?, ?)`,
							genID, nullStr(taskID), urlStr, localPath, time.Now())
						downloaded = 1
						runtime.LogInfo(a.ctx, fmt.Sprintf("[SaveDraftsAndDownload] 已下载: %s (task_id=%s)", localPath, taskID))
					}
				} else {
					resp.Body.Close()
				}
			} else if resp != nil {
				resp.Body.Close()
			}
		}
	}
	runtime.LogInfo(a.ctx, fmt.Sprintf("[SaveDraftsAndDownload] 共下载 %d 个视频到 %s", downloaded, downloadDir))
	return jsonMarshal(map[string]interface{}{
		"success":     true,
		"message":     fmt.Sprintf("已下载 %d 个视频到 %s", downloaded, downloadDir),
		"downloaded":  downloaded,
		"download_dir": downloadDir,
	})
}

// ClearVideoDownloads 清空 video_downloads 表并删除 downloads 文件夹下所有文件（用于纠错或重置）
func (a *App) ClearVideoDownloads() (string, error) {
	baseDir, err := os.Getwd()
	if err != nil {
		baseDir = "."
	}
	downloadDir := filepath.Join(baseDir, "downloads")
	removed := 0
	if entries, err := os.ReadDir(downloadDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				p := filepath.Join(downloadDir, e.Name())
				if err := os.Remove(p); err == nil {
					removed++
				}
			}
		}
	}
	if a.db != nil {
		_, _ = a.db.Exec(`DELETE FROM video_downloads`)
	}
	runtime.LogInfo(a.ctx, fmt.Sprintf("[ClearVideoDownloads] 已删除 %d 个文件并清空 video_downloads 表", removed))
	return jsonMarshal(map[string]interface{}{"success": true, "removed_files": removed})
}

// DeleteTaskData 删除指定 task_id 的数据库记录，可选删除本地文件
// deleteFile=true 时删除 video_downloads.local_path 对应文件
func (a *App) DeleteTaskData(taskId string, deleteFile bool) (string, error) {
	taskId = strings.TrimSpace(taskId)
	if taskId == "" {
		return jsonFail("task_id 不能为空")
	}
	if a.db == nil {
		return jsonMarshal(map[string]interface{}{"success": true})
	}
	if deleteFile {
		var localPath sql.NullString
		_ = a.db.QueryRow(`SELECT local_path FROM video_downloads WHERE task_id=?`, taskId).Scan(&localPath)
		if localPath.Valid {
			_ = os.Remove(localPath.String)
		}
	}
	_, _ = a.db.Exec(`DELETE FROM video_downloads WHERE task_id=?`, taskId)
	_, _ = a.db.Exec(`DELETE FROM video_task_results WHERE task_id=?`, taskId)
	return jsonMarshal(map[string]interface{}{"success": true})
}

// ReDownloadVideo 根据 task_id 重新下载视频并返回可用信息
func (a *App) ReDownloadVideo(taskId string) (string, error) {
	taskId = strings.TrimSpace(taskId)
	if taskId == "" {
		return jsonFail("task_id 不能为空")
	}
	if a.db == nil {
		return jsonFail("数据库不可用")
	}
	var urlStr, localPath, genID string
	if err := a.db.QueryRow(`SELECT downloadable_url, local_path, generation_id FROM video_downloads WHERE task_id=? ORDER BY created_at DESC LIMIT 1`, taskId).
		Scan(&urlStr, &localPath, &genID); err != nil {
		return jsonFail("未找到该任务的下载记录")
	}
	urlStr = strings.TrimSpace(urlStr)
	if urlStr == "" {
		return jsonFail("downloadable_url 为空")
	}
	baseDir, err := os.Getwd()
	if err != nil {
		baseDir = "."
	}
	downloadDir := filepath.Join(baseDir, "downloads")
	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		return jsonFail("创建下载目录失败: " + err.Error())
	}
	if strings.TrimSpace(localPath) == "" {
		if strings.TrimSpace(genID) == "" {
			return jsonFail("缺少本地路径和 generation_id")
		}
		localPath = filepath.Join(downloadDir, strings.TrimSpace(genID)+".mp4")
	}
	client := &http.Client{Timeout: 120 * time.Second}
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return jsonFail("创建下载请求失败: " + err.Error())
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	resp, err := client.Do(req)
	if err != nil {
		return jsonFail("下载失败: " + err.Error())
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return jsonFail(fmt.Sprintf("下载失败: HTTP %d", resp.StatusCode))
	}
	f, err := os.Create(localPath)
	if err != nil {
		resp.Body.Close()
		return jsonFail("创建文件失败: " + err.Error())
	}
	_, err = io.Copy(f, resp.Body)
	resp.Body.Close()
	f.Close()
	if err != nil {
		_ = os.Remove(localPath)
		return jsonFail("写入文件失败: " + err.Error())
	}
	_, _ = a.db.Exec(`UPDATE video_downloads SET local_path=?, created_at=? WHERE task_id=?`, localPath, time.Now(), taskId)
	return jsonMarshal(map[string]interface{}{
		"success":          true,
		"local_path":       localPath,
		"downloadable_url": urlStr,
	})
}

func (a *App) simplePostJSON(urlStr string, body map[string]interface{}) (map[string]interface{}, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", urlStr, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(raw))
	}
	var out map[string]interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return map[string]interface{}{"raw": string(raw)}, nil
	}
	return out, nil
}

func extractAnyURL(m map[string]interface{}) string {
	if m == nil {
		return ""
	}
	for _, k := range []string{"url", "download_link", "downloadable_url", "published_url", "video_url", "link"} {
		if v, ok := m[k].(string); ok && strings.TrimSpace(v) != "" {
			return v
		}
	}
	for _, k := range []string{"data", "result", "item"} {
		if sub, ok := m[k].(map[string]interface{}); ok {
			if v := extractAnyURL(sub); v != "" {
				return v
			}
		}
	}
	if items, ok := m["items"].([]interface{}); ok {
		for _, it := range items {
			if sub, ok := it.(map[string]interface{}); ok {
				if v := extractAnyURL(sub); v != "" {
					return v
				}
			}
		}
	}
	return ""
}

func looksLikeDirectMediaURL(u string) bool {
	lu := strings.ToLower(u)
	return strings.Contains(lu, "videos.openai.com") || strings.Contains(lu, ".mp4") || strings.Contains(lu, "/raw")
}

func downloadToFile(urlStr string, localPath string) error {
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	f, err := os.Create(localPath)
	if err != nil {
		resp.Body.Close()
		return err
	}
	_, err = io.Copy(f, resp.Body)
	resp.Body.Close()
	f.Close()
	if err != nil {
		_ = os.Remove(localPath)
		return err
	}
	return nil
}

func logSafeJSON(ctx context.Context, prefix string, v interface{}) {
	b, err := json.Marshal(v)
	if err != nil {
		runtime.LogInfo(ctx, prefix+"<json marshal failed>")
		return
	}
	s := strings.ReplaceAll(string(b), "%", "%%")
	runtime.LogInfo(ctx, prefix+s)
}

func (a *App) fetchPublishedShareURL(apiBaseURL, bearer, taskId, generationID string) (string, string, error) {
	apiBaseURL = strings.TrimRight(strings.TrimSpace(apiBaseURL), "/")
	if apiBaseURL == "" {
		return "", "", fmt.Errorf("apiBaseURL 为空")
	}
	body := map[string]interface{}{
		"bearer_token": bearer,
	}
	if strings.TrimSpace(generationID) != "" {
		body["generation_id"] = generationID
	}
	if strings.TrimSpace(taskId) != "" {
		body["task_id"] = taskId
	}
	runtime.LogInfo(a.ctx, fmt.Sprintf("[PublishNoWM] POST %s/get-published-video-url (fallback)", apiBaseURL))
	resp, err := a.simplePostJSON(apiBaseURL+"/get-published-video-url", body)
	if err != nil {
		return "", "", err
	}
	postID := ""
	if v, ok := resp["post_id"].(string); ok {
		postID = strings.TrimSpace(v)
	}
	if v, ok := resp["share_url"].(string); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v), postID, nil
	}
	return extractAnyURL(resp), postID, nil
}

// PublishAndDownloadNoWatermark 先发布视频，再获取发布地址并解析无水印直链，最后覆盖下载到本地
func (a *App) PublishAndDownloadNoWatermark(apiBaseURL string, taskId string, parseURL string, parseToken string) (string, error) {
	taskId = strings.TrimSpace(taskId)
	if taskId == "" {
		return jsonFail("task_id 不能为空")
	}
	if a.db == nil {
		return jsonFail("数据库不可用")
	}
	apiBaseURL = strings.TrimRight(strings.TrimSpace(apiBaseURL), "/")
	if apiBaseURL == "" {
		apiBaseURL = strings.TrimRight(a.GetBaseURL(), "/")
	}
	if apiBaseURL == "" {
		return jsonFail("apiBaseURL 不能为空")
	}

	var tokenID int64
	var prompt sql.NullString
	if err := a.db.QueryRow(`SELECT token_id, prompt FROM video_task_results WHERE task_id=?`, taskId).Scan(&tokenID, &prompt); err != nil {
		return jsonFail("未找到该任务的 token_id")
	}
	var bearer string
	if err := a.db.QueryRow(`SELECT token FROM tokens WHERE id=?`, tokenID).Scan(&bearer); err != nil {
		return jsonFail("未找到该任务的 bearer token")
	}
	bearer = strings.TrimSpace(bearer)
	if bearer == "" {
		return jsonFail("bearer token 为空")
	}

	var generationID, localPath string
	if err := a.db.QueryRow(`SELECT generation_id, local_path FROM video_downloads WHERE task_id=? ORDER BY created_at DESC LIMIT 1`, taskId).
		Scan(&generationID, &localPath); err != nil {
		return jsonFail("未找到该任务的 generation_id")
	}
	generationID = strings.TrimSpace(generationID)
	if generationID == "" {
		return jsonFail("generation_id 为空")
	}

	// 1) publish-video
	publishBody := map[string]interface{}{
		"bearer_token":  bearer,
		"generation_id": generationID,
		"prompt":        strings.TrimSpace(prompt.String),
	}
	runtime.LogInfo(a.ctx, fmt.Sprintf("[PublishNoWM] POST %s/publish-video (generation_id=%s)", apiBaseURL, generationID))
	pubResp, err := a.simplePostJSON(apiBaseURL+"/publish-video", publishBody)
	logSafeJSON(a.ctx, "[PublishNoWM] publish-video 响应: ", pubResp)
	publishFailed := false
	if err != nil {
		publishFailed = true
	}
	if !publishFailed {
		if v, ok := pubResp["error"]; ok && v != nil {
			publishFailed = true
		}
		if v, ok := pubResp["message"].(string); ok && strings.Contains(v, "409") {
			publishFailed = true
		}
	}
	if publishFailed {
		runtime.LogInfo(a.ctx, "[PublishNoWM] publish-video 失败，继续后续流程（不作为最终失败）")
	}

	// 2) get-published-video-url
	getURLBody := map[string]interface{}{
		"bearer_token": bearer,
		"task_id":      taskId,
	}
	var publishedURL string
	postID := ""
	if !publishFailed {
		runtime.LogInfo(a.ctx, fmt.Sprintf("[PublishNoWM] POST %s/get-published-video-url (task_id=%s)", apiBaseURL, taskId))
		publishedResp, err := a.simplePostJSON(apiBaseURL+"/get-published-video-url", getURLBody)
		if err == nil {
			logSafeJSON(a.ctx, "[PublishNoWM] get-published-video-url 响应: ", publishedResp)
			if v, ok := publishedResp["post_id"].(string); ok {
				postID = strings.TrimSpace(v)
			}
			if v, ok := publishedResp["share_url"].(string); ok && strings.TrimSpace(v) != "" {
				publishedURL = strings.TrimSpace(v)
			} else {
				publishedURL = extractAnyURL(publishedResp)
			}
			if publishedURL == "" {
				if status, ok := publishedResp["status"].(string); ok && status == "pending" {
					publishedURL = ""
				}
			}
		}
	}
	if publishedURL == "" {
	// 发布失败或拿不到发布地址时，改用本地服务 /get-published-video-url 获取 share_url
	runtime.LogInfo(a.ctx, "[PublishNoWM] 未拿到发布地址，尝试通过本地服务 /get-published-video-url 获取 share_url")
	shareURL, pid, err := a.fetchPublishedShareURL(apiBaseURL, bearer, taskId, generationID)
	if err == nil && shareURL != "" {
		publishedURL = shareURL
		if postID == "" {
			postID = pid
		}
		runtime.LogInfo(a.ctx, fmt.Sprintf("[PublishNoWM] 使用 share_url: %s", shareURL))
	}
	}
	if publishedURL == "" {
		return jsonFail("未解析到发布地址")
	}

	// 3) 解析无水印直链
	noWmURL := ""
	if looksLikeDirectMediaURL(publishedURL) {
		noWmURL = publishedURL
	} else {
		parseURL = strings.TrimSpace(parseURL)
		parseToken = strings.TrimSpace(parseToken)
		if parseURL == "" {
			parseURL = "https://api.sorai.me/get-sora-link"
		}
		if parseToken == "" {
			return jsonFail("无水印解析 token 为空")
		}
		runtime.LogInfo(a.ctx, fmt.Sprintf("[PublishNoWM] POST %s (get-sora-link)", parseURL))
		parseResp, err := a.simplePostJSON(parseURL, map[string]interface{}{
			"url":   publishedURL,
			"token": parseToken,
		})
		if err != nil {
			return jsonFail("解析无水印失败: " + err.Error())
		}
		logSafeJSON(a.ctx, "[PublishNoWM] get-sora-link 响应: ", parseResp)
		noWmURL = extractAnyURL(parseResp)
		if noWmURL == "" {
			return jsonFail("未解析到无水印直链")
		}
	}

	// 4) 下载覆盖本地文件
	baseDir, err := os.Getwd()
	if err != nil {
		baseDir = "."
	}
	downloadDir := filepath.Join(baseDir, "downloads")
	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		return jsonFail("创建下载目录失败: " + err.Error())
	}
	if strings.TrimSpace(localPath) == "" {
		localPath = filepath.Join(downloadDir, generationID+".mp4")
	}
	if err := downloadToFile(noWmURL, localPath); err != nil {
		return jsonFail("下载无水印视频失败: " + err.Error())
	}
	runtime.LogInfo(a.ctx, fmt.Sprintf("[PublishNoWM] 已下载并覆盖: %s", localPath))
	if postID == "" {
		postID = publishedURL
	}
	_, _ = a.db.Exec(`UPDATE video_downloads SET local_path=?, downloadable_url=?, post_id=?, created_at=? WHERE task_id=?`, localPath, noWmURL, postID, time.Now(), taskId)

	return jsonMarshal(map[string]interface{}{
		"success":        true,
		"published_url":  publishedURL,
		"no_watermark":   noWmURL,
		"local_path":     localPath,
	})
}

// GetTaskList 从 SQLite 读取任务列表 JSON（key="list"），并合并本地下载路径
// 若 task_list 为空，则回退到 video_task_results 生成占位任务，便于查看已完成任务
func (a *App) GetTaskList() (string, error) {
	if a.db == nil {
		return "[]", nil
	}
	downloads := map[string]string{}
	if drows, derr := a.db.Query(`SELECT task_id, local_path FROM video_downloads WHERE task_id IS NOT NULL`); derr == nil {
		for drows.Next() {
			var taskID string
			var localPath string
			if err := drows.Scan(&taskID, &localPath); err == nil {
				downloads[taskID] = localPath
			}
		}
		drows.Close()
	}

	var value string
	if err := a.db.QueryRow(`SELECT value FROM task_list WHERE key='list'`).Scan(&value); err == nil {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" && trimmed != "[]" && trimmed != "null" {
			var list []map[string]interface{}
			if err := json.Unmarshal([]byte(value), &list); err == nil {
				for i := range list {
					key := ""
					if v, ok := list[i]["remoteTaskId"].(string); ok && v != "" {
						key = v
					} else if v, ok := list[i]["id"].(string); ok && v != "" {
						key = v
					}
					if key != "" {
						if lp, ok := downloads[key]; ok && strings.TrimSpace(lp) != "" {
							list[i]["localPath"] = lp
						}
					}
				}
				runtime.LogInfo(a.ctx, fmt.Sprintf("[GetTaskList] task_list 合并 localPath: %d 条", len(list)))
				if len(list) > 0 {
					runtime.LogInfo(a.ctx, fmt.Sprintf("[GetTaskList] 示例: id=%v localPath=%v", list[0]["id"], list[0]["localPath"]))
				}
				return jsonMarshal(list)
			}
			return value, nil
		}
	}

	rows, err := a.db.Query(`SELECT task_id, token_id, result_json, progress_pct, prompt, created_at FROM video_task_results ORDER BY created_at DESC`)
	if err != nil {
		return "[]", nil
	}
	defer rows.Close()
	var list []map[string]interface{}
	for rows.Next() {
		var taskID string
		var tokenID int64
		var resultJSON sql.NullString
		var progressPct sql.NullFloat64
		var prompt sql.NullString
		var createdAt sql.NullString
		if err := rows.Scan(&taskID, &tokenID, &resultJSON, &progressPct, &prompt, &createdAt); err != nil {
			continue
		}
		pct := 0.0
		if progressPct.Valid {
			pct = progressPct.Float64
		}
		status := "running"
		if pct >= 100 {
			status = "done"
		}
		promptText := ""
		if prompt.Valid {
			promptText = strings.TrimSpace(prompt.String)
		}
		if promptText == "" && taskID != "" {
			promptText = "临时提示词（待补充）"
			_, _ = a.db.Exec(`UPDATE video_task_results SET prompt=? WHERE task_id=?`, promptText, taskID)
		}
		localPath := downloads[taskID]
		createdAtVal := ""
		if createdAt.Valid {
			createdAtVal = createdAt.String
		}
		list = append(list, map[string]interface{}{
			"id":               taskID,
			"model":            "sora2-unknown",
			"prompt":           promptText,
			"status":           status,
			"progress":         pct,
			"message":          "来自数据库",
			"remoteTaskId":     taskID,
			"tokenIdForPending": tokenID,
			"result":           resultJSON.String,
			"localPath":        localPath,
			"timestamp":        createdAtVal,
		})
	}
	if len(list) == 0 {
		runtime.LogInfo(a.ctx, "[GetTaskList] 回退 video_task_results: 0 条")
		return "[]", nil
	}
	runtime.LogInfo(a.ctx, fmt.Sprintf("[GetTaskList] 回退 video_task_results: %d 条", len(list)))
	return jsonMarshal(list)
}

// SetTaskList 将任务列表 JSON 写入 SQLite（key="list"）
func (a *App) SetTaskList(jsonStr string) (string, error) {
	if a.db == nil {
		return jsonMarshal(map[string]interface{}{"success": true})
	}
	_, err := a.db.Exec(`INSERT OR REPLACE INTO task_list (key, value) VALUES ('list', ?)`, jsonStr)
	if err != nil {
		return jsonFail("写入任务列表失败: " + err.Error())
	}
	return jsonMarshal(map[string]interface{}{"success": true})
}

// GetVideoDownloadsMap 返回 task_id -> local_path 的映射，用于前端显示本地预览
func (a *App) GetVideoDownloadsMap() (string, error) {
	if a.db == nil {
		return jsonMarshal(map[string]interface{}{"map": map[string]string{}})
	}
	rows, err := a.db.Query(`SELECT task_id, local_path FROM video_downloads WHERE task_id IS NOT NULL`)
	if err != nil {
		return jsonFail("查询 video_downloads 失败: " + err.Error())
	}
	defer rows.Close()
	m := map[string]string{}
	for rows.Next() {
		var taskID string
		var localPath string
		if err := rows.Scan(&taskID, &localPath); err != nil {
			continue
		}
		if strings.TrimSpace(taskID) != "" && strings.TrimSpace(localPath) != "" {
			m[taskID] = localPath
		}
	}
	return jsonMarshal(map[string]interface{}{"map": m})
}

// GetLocalFileDataURL 读取本地文件并返回 data URL（用于前端预览本地 MP4）
func (a *App) GetLocalFileDataURL(path string) (string, error) {
	p := strings.TrimSpace(path)
	if p == "" {
		return "", fmt.Errorf("path 不能为空")
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}
	ext := strings.ToLower(filepath.Ext(p))
	mime := "application/octet-stream"
	if ext == ".mp4" {
		mime = "video/mp4"
	}
	encoded := base64.StdEncoding.EncodeToString(data)
	return "data:" + mime + ";base64," + encoded, nil
}

func (a *App) ensureLocalFileServer() (int, error) {
	var startErr error
	a.fileServerOnce.Do(func() {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			startErr = err
			return
		}
		a.fileServerPort = ln.Addr().(*net.TCPAddr).Port

		mux := http.NewServeMux()
		mux.HandleFunc("/localfile", func(w http.ResponseWriter, r *http.Request) {
			raw := r.URL.Query().Get("path")
			if raw == "" {
				http.Error(w, "path required", http.StatusBadRequest)
				return
			}
			p, _ := url.QueryUnescape(raw)
			p = strings.TrimSpace(p)
			if p == "" {
				http.Error(w, "path required", http.StatusBadRequest)
				return
			}
			baseDir, err := os.Getwd()
			if err != nil {
				baseDir = "."
			}
			downloadDir := filepath.Join(baseDir, "downloads")
			absPath, err := filepath.Abs(p)
			if err != nil {
				http.Error(w, "invalid path", http.StatusBadRequest)
				return
			}
			absDownload, _ := filepath.Abs(downloadDir)
			rel, err := filepath.Rel(absDownload, absPath)
			if err != nil || strings.HasPrefix(rel, "..") {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Accept-Ranges", "bytes")
			http.ServeFile(w, r, absPath)
		})

		go func() {
			_ = http.Serve(ln, mux)
		}()
	})
	return a.fileServerPort, startErr
}

// GetLocalFileURL 返回本地文件的可访问 URL（流式播放）
func (a *App) GetLocalFileURL(path string) (string, error) {
	p := strings.TrimSpace(path)
	if p == "" {
		return "", fmt.Errorf("path 不能为空")
	}
	port, err := a.ensureLocalFileServer()
	if err != nil {
		return "", err
	}
	u := url.URL{
		Scheme:   "http",
		Host:     fmt.Sprintf("127.0.0.1:%d", port),
		Path:     "/localfile",
		RawQuery: "path=" + url.QueryEscape(p),
	}
	return u.String(), nil
}

func (a *App) localTokenSetStatus(id int64, body string) (string, error) {
	var input struct {
		IsActive bool `json:"is_active"`
	}
	_ = json.Unmarshal([]byte(body), &input)
	return a.localTokenSetActive(id, input.IsActive)
}

func (a *App) localTokensImport(body string) (string, error) {
	var input struct {
		Tokens []string `json:"tokens"`
		Mode   string   `json:"mode"`
	}
	if err := json.Unmarshal([]byte(body), &input); err != nil {
		return jsonFail("请求体解析失败")
	}
	now := time.Now()
	added := 0
	for _, t := range input.Tokens {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		_, err := a.db.Exec(
			`INSERT INTO tokens (token, is_active, created_at, updated_at) VALUES (?, 1, ?, ?)`,
			t, now, now,
		)
		if err != nil {
			continue
		}
		added++
	}
	return jsonMarshal(map[string]interface{}{"success": true, "message": fmt.Sprintf("成功导入 %d 个 Token", added), "imported": added})
}

func (a *App) localTokensBatch(method string, parts []string, body string) (string, error) {
	var tokenIDs []int64
	if body != "" {
		var input struct {
			TokenIDs []int64 `json:"token_ids"`
		}
		if err := json.Unmarshal([]byte(body), &input); err == nil {
			tokenIDs = input.TokenIDs
		}
	}
	if len(parts) < 3 {
		return jsonFail("缺少 batch 操作类型")
	}
	action := parts[2]
	now := time.Now()

	switch action {
	case "test-update":
		for _, id := range tokenIDs {
			_, _ = a.localTokenTest(id)
		}
		return jsonMarshal(map[string]interface{}{"success": true, "message": "批量测试完成"})
	case "enable-all":
		for _, id := range tokenIDs {
			_, _ = a.db.Exec(`UPDATE tokens SET is_active=1, updated_at=? WHERE id=?`, now, id)
		}
		return jsonMarshal(map[string]interface{}{"success": true, "message": "已批量启用"})
	case "disable-selected":
		for _, id := range tokenIDs {
			_, _ = a.db.Exec(`UPDATE tokens SET is_active=0, updated_at=? WHERE id=?`, now, id)
		}
		return jsonMarshal(map[string]interface{}{"success": true, "message": "已批量禁用"})
	case "delete-disabled":
		for _, id := range tokenIDs {
			_, _ = a.db.Exec(`DELETE FROM tokens WHERE id=?`, id)
		}
		return jsonMarshal(map[string]interface{}{"success": true, "message": "已批量删除"})
	case "update-proxy":
		var input struct {
			TokenIDs []int64 `json:"token_ids"`
			ProxyURL string  `json:"proxy_url"`
		}
		if err := json.Unmarshal([]byte(body), &input); err != nil {
			return jsonFail("请求体解析失败")
		}
		for _, id := range input.TokenIDs {
			_, _ = a.db.Exec(`UPDATE tokens SET proxy_url=?, updated_at=? WHERE id=?`, input.ProxyURL, now, id)
		}
		return jsonMarshal(map[string]interface{}{"success": true, "message": "代理已更新"})
	default:
		return jsonFail("未知的 batch 操作: " + action)
	}
}

func (a *App) handleLocalAdmin(method string, path string, parts []string, body string) (string, error) {
	if len(parts) < 2 {
		return jsonMarshal(map[string]interface{}{
			"admin_username": "", "proxy_enabled": false, "proxy_url": "", "error_ban_threshold": 5,
			"cache_enabled": true, "cache_timeout": 7200, "cache_base_url": "", "image_timeout": 300, "video_timeout": 1500,
			"debug_enabled": false, "watermark_enabled": true,
		})
	}
	switch parts[1] {
	case "config":
		if method == http.MethodGet {
			return jsonMarshal(map[string]interface{}{
				"admin_username": "", "proxy_enabled": false, "proxy_url": "", "error_ban_threshold": 5,
				"cache_enabled": true, "cache_timeout": 7200, "cache_base_url": "", "image_timeout": 300, "video_timeout": 1500,
				"debug_enabled": false, "watermark_enabled": true,
			})
		}
		if method == http.MethodPost {
			return jsonMarshal(map[string]interface{}{"success": true})
		}
	case "password", "apikey", "debug":
		return jsonMarshal(map[string]interface{}{"success": true})
	}
	return jsonMarshal(map[string]interface{}{})
}

func (a *App) handleLocalLogs(method string, path string, body string) (string, error) {
	if method == http.MethodGet {
		return jsonMarshal(map[string]interface{}{"logs": []interface{}{}, "total": 0})
	}
	if method == http.MethodDelete {
		return jsonMarshal(map[string]interface{}{"success": true})
	}
	return jsonMarshal(map[string]interface{}{"logs": []interface{}{}, "total": 0})
}

func (a *App) handleLocalStats(method string, path string) (string, error) {
	total := 0
	active := 0
	if a.db != nil {
		_ = a.db.QueryRow(`SELECT COUNT(*) FROM tokens`).Scan(&total)
		_ = a.db.QueryRow(`SELECT COUNT(*) FROM tokens WHERE is_active=1`).Scan(&active)
	}
	return jsonMarshal(map[string]interface{}{
		"total_tokens":   total,
		"active_tokens":  active,
		"today_images":   0, "total_images": 0, "today_videos": 0, "total_videos": 0, "today_errors": 0, "total_errors": 0,
	})
}

func (a *App) handleLocalConfigDefault(prefix string, method string, path string, body string) (string, error) {
	switch prefix {
	case "proxy":
		if method == http.MethodGet {
			return jsonMarshal(map[string]interface{}{"proxy_enabled": false, "proxy_url": ""})
		}
		return jsonMarshal(map[string]interface{}{"success": true})
	case "watermark-free":
		if method == http.MethodGet {
			enabled := false
			parseMethod := "third_party"
			customURL := ""
			customToken := ""
			if a.db != nil {
				enabled = strings.TrimSpace(a.getSettingValue("watermark_free_enabled")) == "true"
				if v := strings.TrimSpace(a.getSettingValue("watermark_parse_method")); v != "" {
					parseMethod = v
				}
				customURL = a.getSettingValue("watermark_custom_url")
				customToken = a.getSettingValue("watermark_custom_token")
			}
			return jsonMarshal(map[string]interface{}{
				"watermark_free_enabled": enabled,
				"parse_method":           parseMethod,
				"custom_parse_url":       customURL,
				"custom_parse_token":     customToken,
			})
		}
		if method == http.MethodPost {
			var input struct {
				WatermarkFreeEnabled bool   `json:"watermark_free_enabled"`
				ParseMethod          string `json:"parse_method"`
				CustomParseURL       string `json:"custom_parse_url"`
				CustomParseToken     string `json:"custom_parse_token"`
			}
			if err := json.Unmarshal([]byte(body), &input); err != nil {
				return jsonFail("请求体解析失败")
			}
			if a.db != nil {
				a.setSettingValue("watermark_free_enabled", strconv.FormatBool(input.WatermarkFreeEnabled))
				a.setSettingValue("watermark_parse_method", strings.TrimSpace(input.ParseMethod))
				a.setSettingValue("watermark_custom_url", strings.TrimSpace(input.CustomParseURL))
				a.setSettingValue("watermark_custom_token", strings.TrimSpace(input.CustomParseToken))
				runtime.LogInfo(a.ctx, fmt.Sprintf("[WatermarkConfig] 保存: enabled=%v method=%s url=%s tokenLen=%d", input.WatermarkFreeEnabled, input.ParseMethod, input.CustomParseURL, len(input.CustomParseToken)))
			}
			return jsonMarshal(map[string]interface{}{"success": true})
		}
		return jsonMarshal(map[string]interface{}{"success": true})
	case "cache":
		if method == http.MethodGet {
			return jsonMarshal(map[string]interface{}{"config": map[string]interface{}{"enabled": true, "timeout": 7200, "base_url": "", "effective_base_url": ""}})
		}
		return jsonMarshal(map[string]interface{}{"success": true})
	case "generation":
		if method == http.MethodGet {
			return jsonMarshal(map[string]interface{}{"config": map[string]interface{}{"image_timeout": 300, "video_timeout": 1500}})
		}
		return jsonMarshal(map[string]interface{}{"success": true})
	case "token-refresh":
		if method == http.MethodGet {
			return jsonMarshal(map[string]interface{}{"success": true, "config": map[string]interface{}{"at_auto_refresh_enabled": false}})
		}
		return jsonMarshal(map[string]interface{}{"success": true})
	}
	return jsonMarshal(map[string]interface{}{})
}

func (a *App) handleLocalTasks(method string, path string, parts []string, body string) (string, error) {
	return jsonMarshal(map[string]interface{}{"success": true, "message": "本地模式下任务取消请在前端处理"})
}

// ApiRequestBlob 用于下载文件等二进制内容，返回 base64 编码的字符串
func (a *App) ApiRequestBlob(method string, path string, token string) (string, error) {
	base := strings.TrimRight(a.GetBaseURL(), "/")
	fullURL := base + path

	req, err := http.NewRequest(method, fullURL, nil)
	if err != nil {
		return "", err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return fmt.Sprintf("data:application/octet-stream;base64,%s",
		base64Encode(respBody)), nil
}

// TestServerHealth 测试指定或当前服务器的 /health 接口
func (a *App) TestServerHealth(baseURL string) (*HealthResult, error) {
	base := strings.TrimSpace(baseURL)
	if base == "" {
		base = a.GetBaseURL()
	}
	base = strings.TrimRight(base, "/")
	testURL := base + "/health"

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest(http.MethodGet, testURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	runtime.LogInfo(a.ctx, fmt.Sprintf("测试服务器健康状态: %s", testURL))

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return &HealthResult{
			Ok:      false,
			Message: fmt.Sprintf("服务器响应异常 (HTTP %d): %s", resp.StatusCode, string(body)),
		}, nil
	}

	// 尝试解析 {"status":"ok"}
	var parsed struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(body, &parsed); err == nil && strings.ToLower(parsed.Status) == "ok" {
		return &HealthResult{
			Ok:      true,
			Message: "服务器连接正常，可以使用",
		}, nil
	}

	// 200 但内容不标准，也认为连通，只是提示用户手动确认
	return &HealthResult{
		Ok:      true,
		Message: "服务器已连接，但返回内容非标准格式，请手动确认服务是否正常",
	}, nil
}

func base64Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// UpdateInfo 更新信息结构
type UpdateInfo struct {
	HasUpdate   bool   `json:"has_update"`
	LatestVersion string `json:"latest_version"`
	CurrentVersion string `json:"current_version"`
	DownloadURL  string `json:"download_url"`
	ReleaseNotes string `json:"release_notes"`
	Error        string `json:"error,omitempty"`
}

// GetCurrentVersion 返回当前应用版本
func (a *App) GetCurrentVersion() string {
	return AppVersion
}

// CheckForUpdates 检查 GitHub Releases 是否有新版本
func (a *App) CheckForUpdates() (string, error) {
	currentVersion := AppVersion
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", GitHubOwner, GitHubRepo)
	
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return jsonMarshal(UpdateInfo{
			HasUpdate:      false,
			CurrentVersion: currentVersion,
			Error:          "创建请求失败: " + err.Error(),
		})
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "sorapc-updater")
	
	resp, err := client.Do(req)
	if err != nil {
		return jsonMarshal(UpdateInfo{
			HasUpdate:      false,
			CurrentVersion: currentVersion,
			Error:          "网络请求失败: " + err.Error(),
		})
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return jsonMarshal(UpdateInfo{
			HasUpdate:      false,
			CurrentVersion: currentVersion,
			Error:          fmt.Sprintf("GitHub API 返回错误: HTTP %d", resp.StatusCode),
		})
	}
	
	var release struct {
		TagName    string `json:"tag_name"`
		Name       string `json:"name"`
		Body       string `json:"body"`
		Assets     []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return jsonMarshal(UpdateInfo{
			HasUpdate:      false,
			CurrentVersion: currentVersion,
			Error:          "读取响应失败: " + err.Error(),
		})
	}
	
	if err := json.Unmarshal(body, &release); err != nil {
		return jsonMarshal(UpdateInfo{
			HasUpdate:      false,
			CurrentVersion: currentVersion,
			Error:          "解析响应失败: " + err.Error(),
		})
	}
	
	latestVersion := strings.TrimPrefix(release.TagName, "v")
	hasUpdate := compareVersions(latestVersion, currentVersion) > 0
	
	// 查找对应平台的安装包
	downloadURL := ""
	platform := goruntime.GOOS
	arch := goruntime.GOARCH
	
	for _, asset := range release.Assets {
		assetName := strings.ToLower(asset.Name)
		
		// Windows 平台
		if platform == "windows" {
			if strings.HasSuffix(assetName, ".exe") || 
			   strings.HasSuffix(assetName, ".msi") ||
			   strings.Contains(assetName, "windows") {
				downloadURL = asset.BrowserDownloadURL
				break
			}
		}
		
		// macOS 平台
		if platform == "darwin" {
		// 优先选择对应架构的版本
		if arch == "arm64" && strings.Contains(assetName, "arm64") {
				downloadURL = asset.BrowserDownloadURL
				break
			}
			if arch == "amd64" && strings.Contains(assetName, "amd64") {
				downloadURL = asset.BrowserDownloadURL
				break
			}
			// 如果没有找到对应架构，选择任何 macOS 版本
			if downloadURL == "" && (strings.Contains(assetName, "macos") || strings.HasSuffix(assetName, ".app") || strings.HasSuffix(assetName, ".zip")) {
				downloadURL = asset.BrowserDownloadURL
			}
		}
	}
	
	// 如果没有找到特定平台的包，使用第一个资源
	if downloadURL == "" && len(release.Assets) > 0 {
		downloadURL = release.Assets[0].BrowserDownloadURL
	}
	
	updateInfo := UpdateInfo{
		HasUpdate:      hasUpdate,
		LatestVersion:  latestVersion,
		CurrentVersion: currentVersion,
		DownloadURL:    downloadURL,
		ReleaseNotes:   release.Body,
	}
	
	return jsonMarshal(updateInfo)
}

// compareVersions 比较两个版本号，返回: 1 如果 v1 > v2, -1 如果 v1 < v2, 0 如果相等
func compareVersions(v1, v2 string) int {
	parts1 := strings.Split(strings.TrimPrefix(v1, "v"), ".")
	parts2 := strings.Split(strings.TrimPrefix(v2, "v"), ".")
	
	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}
	
	for i := 0; i < maxLen; i++ {
		var num1, num2 int
		if i < len(parts1) {
			num1, _ = strconv.Atoi(parts1[i])
		}
		if i < len(parts2) {
			num2, _ = strconv.Atoi(parts2[i])
		}
		
		if num1 > num2 {
			return 1
		}
		if num1 < num2 {
			return -1
		}
	}
	return 0
}

// DownloadUpdate 下载更新文件
func (a *App) DownloadUpdate(downloadURL string) (string, error) {
	if downloadURL == "" {
		return jsonFail("下载地址为空")
	}
	
	baseDir, err := os.Getwd()
	if err != nil {
		baseDir = "."
	}
	downloadDir := filepath.Join(baseDir, "updates")
	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		return jsonFail("创建下载目录失败: " + err.Error())
	}
	
	// 从 URL 提取文件名
	u, err := url.Parse(downloadURL)
	if err != nil {
		return jsonFail("无效的下载地址: " + err.Error())
	}
	
	fileName := filepath.Base(u.Path)
	if fileName == "" || fileName == "/" {
		fileName = "update.exe"
	}
	
	localPath := filepath.Join(downloadDir, fileName)
	
	runtime.LogInfo(a.ctx, fmt.Sprintf("开始下载更新: %s -> %s", downloadURL, localPath))
	
	client := &http.Client{
		Timeout: 300 * time.Second, // 5分钟超时
	}
	
	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return jsonFail("创建下载请求失败: " + err.Error())
	}
	req.Header.Set("User-Agent", "sorapc-updater")
	
	resp, err := client.Do(req)
	if err != nil {
		return jsonFail("下载失败: " + err.Error())
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return jsonFail(fmt.Sprintf("下载失败: HTTP %d", resp.StatusCode))
	}
	
	// 创建文件
	file, err := os.Create(localPath)
	if err != nil {
		return jsonFail("创建文件失败: " + err.Error())
	}
	defer file.Close()
	
	// 写入文件
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		os.Remove(localPath)
		return jsonFail("写入文件失败: " + err.Error())
	}
	
	runtime.LogInfo(a.ctx, fmt.Sprintf("更新文件下载完成: %s", localPath))
	
	return jsonMarshal(map[string]interface{}{
		"success":    true,
		"local_path": localPath,
		"message":    "更新文件下载完成",
	})
}

// InstallUpdate 安装更新（Windows/macOS 上执行安装程序）
func (a *App) InstallUpdate(installerPath string) (string, error) {
	if installerPath == "" {
		return jsonFail("安装程序路径为空")
	}
	
	// 检查文件是否存在
	if _, err := os.Stat(installerPath); os.IsNotExist(err) {
		return jsonFail("安装程序文件不存在: " + installerPath)
	}
	
	runtime.LogInfo(a.ctx, fmt.Sprintf("准备安装更新: %s", installerPath))
	
	var cmd *exec.Cmd
	var message string
	
	if goruntime.GOOS == "windows" {
		// Windows 安装
		if strings.HasSuffix(strings.ToLower(installerPath), ".msi") {
			// MSI 安装包
			cmd = exec.Command("msiexec", "/i", installerPath, "/quiet", "/norestart")
		} else {
			// EXE 安装包 - 使用 /S 静默安装（如果支持）
			cmd = exec.Command(installerPath, "/S")
		}
		message = "安装程序已启动，应用即将关闭"
	} else if goruntime.GOOS == "darwin" {
		// macOS 安装
		if strings.HasSuffix(installerPath, ".zip") {
			// ZIP 文件需要先解压
			extractDir := filepath.Join(filepath.Dir(installerPath), "extracted")
			os.RemoveAll(extractDir)
			os.MkdirAll(extractDir, 0755)
			
			// 使用 unzip 解压
			cmd = exec.Command("unzip", "-q", installerPath, "-d", extractDir)
			if err := cmd.Run(); err != nil {
				return jsonFail("解压失败: " + err.Error())
			}
			
			// 查找 .app 文件
			appPath := ""
			err := filepath.Walk(extractDir, func(path string, info os.FileInfo, err error) error {
				if strings.HasSuffix(path, ".app") {
					appPath = path
					return filepath.SkipAll
				}
				return nil
			})
			
			if appPath == "" || err != nil {
				return jsonFail("未找到 .app 文件")
			}
			
			// 复制到 Applications 目录
			appsDir := "/Applications"
			appName := filepath.Base(appPath)
			targetPath := filepath.Join(appsDir, appName)
			
			// 删除旧版本
			os.RemoveAll(targetPath)
			
			// 复制新版本
			cmd = exec.Command("cp", "-R", appPath, targetPath)
			message = "应用已安装到 /Applications，请手动启动新版本"
		} else if strings.HasSuffix(installerPath, ".app") {
			// 直接是 .app 文件
			appsDir := "/Applications"
			appName := filepath.Base(installerPath)
			targetPath := filepath.Join(appsDir, appName)
			
			os.RemoveAll(targetPath)
			cmd = exec.Command("cp", "-R", installerPath, targetPath)
			message = "应用已安装到 /Applications，请手动启动新版本"
		} else {
			return jsonFail("不支持的 macOS 安装包格式")
		}
	} else {
		return jsonFail("当前系统不支持自动安装")
	}
	
	// 执行安装命令
	err := cmd.Run()
	if err != nil {
		return jsonFail("安装失败: " + err.Error())
	}
	
	runtime.LogInfo(a.ctx, message)
	
	// Windows 上延迟关闭，macOS 上立即提示
	if goruntime.GOOS == "windows" {
		go func() {
			time.Sleep(2 * time.Second)
			runtime.Quit(a.ctx)
		}()
	}
	
	return jsonMarshal(map[string]interface{}{
		"success": true,
		"message": message,
	})
}
