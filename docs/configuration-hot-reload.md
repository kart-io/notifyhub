# 配置热重载方案

## 概述

配置热重载功能允许NotifyHub在运行时动态更新配置，无需重启服务即可应用新的配置参数。这对于生产环境的配置调整、平台参数优化和紧急配置修复非常重要。

## 需求分析

### 热重载场景

1. **平台配置更新**：webhook URL、认证信息变更
2. **性能参数调优**：重试策略、限流参数、队列大小
3. **路由规则调整**：消息路由规则的实时调整
4. **告警配置**：告警阈值和通知规则的动态修改
5. **紧急处理**：快速禁用故障平台或调整策略

### 热重载约束

- **安全性**：配置变更需要权限验证
- **一致性**：确保配置变更的原子性
- **兼容性**：新配置必须向后兼容
- **可回滚**：支持配置回滚到上一版本

## 设计方案

### 1. 配置管理架构

```go
// 配置管理器接口
type ConfigurationManager interface {
    // 配置操作
    GetCurrentConfig() *Config
    UpdateConfig(newConfig *Config) error
    ReloadConfig() error
    ValidateConfig(config *Config) []ConfigError

    // 配置源管理
    AddConfigSource(source ConfigSource) error
    RemoveConfigSource(name string) error
    GetConfigSources() []ConfigSource

    // 配置监听
    WatchConfig(callback ConfigChangeCallback) error
    StopWatching() error

    // 配置历史
    GetConfigHistory() []ConfigSnapshot
    RollbackTo(version string) error

    // 配置diff
    CompareConfigs(old, new *Config) ConfigDiff
}

// 配置源接口
type ConfigSource interface {
    Name() string
    Priority() int
    LoadConfig() (*Config, error)
    WatchChanges(callback func(*Config)) error
    StopWatching() error
    SupportsReload() bool
}

// 配置变更回调
type ConfigChangeCallback func(oldConfig, newConfig *Config, diff ConfigDiff) error

// 配置快照
type ConfigSnapshot struct {
    Version   string    `json:"version"`
    Config    *Config   `json:"config"`
    Timestamp time.Time `json:"timestamp"`
    Source    string    `json:"source"`
    Changes   ConfigDiff `json:"changes"`
}

// 配置差异
type ConfigDiff struct {
    Added    map[string]interface{} `json:"added"`
    Modified map[string]ConfigChange `json:"modified"`
    Removed  map[string]interface{} `json:"removed"`
}

type ConfigChange struct {
    OldValue interface{} `json:"old_value"`
    NewValue interface{} `json:"new_value"`
    Path     string      `json:"path"`
}
```

### 2. 配置源实现

```go
// 文件配置源
type FileConfigSource struct {
    filePath    string
    watcher     *fsnotify.Watcher
    callback    func(*Config)
    lastModTime time.Time
    priority    int
}

func NewFileConfigSource(filePath string, priority int) *FileConfigSource {
    return &FileConfigSource{
        filePath: filePath,
        priority: priority,
    }
}

func (fcs *FileConfigSource) LoadConfig() (*Config, error) {
    data, err := ioutil.ReadFile(fcs.filePath)
    if err != nil {
        return nil, fmt.Errorf("failed to read config file: %w", err)
    }

    var config Config
    if err := yaml.Unmarshal(data, &config); err != nil {
        return nil, fmt.Errorf("failed to parse config: %w", err)
    }

    return &config, nil
}

func (fcs *FileConfigSource) WatchChanges(callback func(*Config)) error {
    watcher, err := fsnotify.NewWatcher()
    if err != nil {
        return err
    }

    fcs.watcher = watcher
    fcs.callback = callback

    go func() {
        for {
            select {
            case event, ok := <-watcher.Events:
                if !ok {
                    return
                }

                if event.Op&fsnotify.Write == fsnotify.Write {
                    // 防抖动：等待写入完成
                    time.Sleep(100 * time.Millisecond)

                    config, err := fcs.LoadConfig()
                    if err != nil {
                        log.Printf("Failed to reload config: %v", err)
                        continue
                    }

                    callback(config)
                }

            case err, ok := <-watcher.Errors:
                if !ok {
                    return
                }
                log.Printf("Config watcher error: %v", err)
            }
        }
    }()

    return watcher.Add(fcs.filePath)
}

// HTTP配置源
type HTTPConfigSource struct {
    url         string
    interval    time.Duration
    client      *http.Client
    credentials HTTPCredentials
    priority    int
    stopChan    chan struct{}
}

type HTTPCredentials struct {
    Username string `json:"username"`
    Password string `json:"password"`
    Token    string `json:"token"`
}

func (hcs *HTTPConfigSource) LoadConfig() (*Config, error) {
    req, err := http.NewRequest("GET", hcs.url, nil)
    if err != nil {
        return nil, err
    }

    // 添加认证信息
    if hcs.credentials.Token != "" {
        req.Header.Set("Authorization", "Bearer "+hcs.credentials.Token)
    } else if hcs.credentials.Username != "" {
        req.SetBasicAuth(hcs.credentials.Username, hcs.credentials.Password)
    }

    resp, err := hcs.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
    }

    var config Config
    if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
        return nil, err
    }

    return &config, nil
}

func (hcs *HTTPConfigSource) WatchChanges(callback func(*Config)) error {
    hcs.stopChan = make(chan struct{})

    go func() {
        ticker := time.NewTicker(hcs.interval)
        defer ticker.Stop()

        var lastETag string

        for {
            select {
            case <-hcs.stopChan:
                return
            case <-ticker.C:
                req, err := http.NewRequest("HEAD", hcs.url, nil)
                if err != nil {
                    continue
                }

                if lastETag != "" {
                    req.Header.Set("If-None-Match", lastETag)
                }

                resp, err := hcs.client.Do(req)
                if err != nil {
                    continue
                }
                resp.Body.Close()

                if resp.StatusCode == http.StatusNotModified {
                    continue
                }

                currentETag := resp.Header.Get("ETag")
                if currentETag != lastETag {
                    config, err := hcs.LoadConfig()
                    if err != nil {
                        log.Printf("Failed to load config: %v", err)
                        continue
                    }

                    lastETag = currentETag
                    callback(config)
                }
            }
        }
    }()

    return nil
}

// Consul配置源
type ConsulConfigSource struct {
    client   *consul.Client
    keyPath  string
    priority int
    stopChan chan struct{}
}

func (ccs *ConsulConfigSource) LoadConfig() (*Config, error) {
    kv := ccs.client.KV()
    pair, _, err := kv.Get(ccs.keyPath, nil)
    if err != nil {
        return nil, err
    }

    if pair == nil {
        return nil, fmt.Errorf("config key not found: %s", ccs.keyPath)
    }

    var config Config
    if err := json.Unmarshal(pair.Value, &config); err != nil {
        return nil, err
    }

    return &config, nil
}

func (ccs *ConsulConfigSource) WatchChanges(callback func(*Config)) error {
    ccs.stopChan = make(chan struct{})

    go func() {
        var lastIndex uint64

        for {
            select {
            case <-ccs.stopChan:
                return
            default:
                kv := ccs.client.KV()
                pair, meta, err := kv.Get(ccs.keyPath, &consul.QueryOptions{
                    WaitIndex: lastIndex,
                    WaitTime:  30 * time.Second,
                })

                if err != nil {
                    log.Printf("Consul watch error: %v", err)
                    time.Sleep(5 * time.Second)
                    continue
                }

                if meta.LastIndex > lastIndex {
                    lastIndex = meta.LastIndex

                    if pair != nil {
                        var config Config
                        if err := json.Unmarshal(pair.Value, &config); err != nil {
                            log.Printf("Failed to unmarshal config: %v", err)
                            continue
                        }

                        callback(&config)
                    }
                }
            }
        }
    }()

    return nil
}
```

### 3. 配置管理器实现

```go
// 配置管理器实现
type DefaultConfigurationManager struct {
    currentConfig   *Config
    configSources   []ConfigSource
    callbacks       []ConfigChangeCallback
    history         []ConfigSnapshot
    validator       ConfigValidator
    mutex           sync.RWMutex
    maxHistorySize  int
}

func NewConfigurationManager(initialConfig *Config) *DefaultConfigurationManager {
    return &DefaultConfigurationManager{
        currentConfig:  initialConfig,
        configSources:  make([]ConfigSource, 0),
        callbacks:      make([]ConfigChangeCallback, 0),
        history:        make([]ConfigSnapshot, 0),
        maxHistorySize: 50,
        validator:      &DefaultConfigValidator{},
    }
}

func (dcm *DefaultConfigurationManager) GetCurrentConfig() *Config {
    dcm.mutex.RLock()
    defer dcm.mutex.RUnlock()

    // 返回配置的深拷贝，防止外部修改
    return dcm.deepCopyConfig(dcm.currentConfig)
}

func (dcm *DefaultConfigurationManager) UpdateConfig(newConfig *Config) error {
    // 验证新配置
    if errors := dcm.validator.Validate(newConfig); len(errors) > 0 {
        return fmt.Errorf("config validation failed: %v", errors)
    }

    dcm.mutex.Lock()
    defer dcm.mutex.Unlock()

    oldConfig := dcm.currentConfig
    diff := dcm.compareConfigs(oldConfig, newConfig)

    // 创建配置快照
    snapshot := ConfigSnapshot{
        Version:   dcm.generateVersion(),
        Config:    dcm.deepCopyConfig(newConfig),
        Timestamp: time.Now(),
        Source:    "manual",
        Changes:   diff,
    }

    // 执行配置变更回调
    for _, callback := range dcm.callbacks {
        if err := callback(oldConfig, newConfig, diff); err != nil {
            return fmt.Errorf("config change callback failed: %w", err)
        }
    }

    // 更新当前配置
    dcm.currentConfig = newConfig

    // 保存到历史记录
    dcm.addToHistory(snapshot)

    return nil
}

func (dcm *DefaultConfigurationManager) ReloadConfig() error {
    // 按优先级排序配置源
    sort.Slice(dcm.configSources, func(i, j int) bool {
        return dcm.configSources[i].Priority() > dcm.configSources[j].Priority()
    })

    // 合并配置
    var mergedConfig *Config
    for _, source := range dcm.configSources {
        config, err := source.LoadConfig()
        if err != nil {
            log.Printf("Failed to load config from %s: %v", source.Name(), err)
            continue
        }

        if mergedConfig == nil {
            mergedConfig = config
        } else {
            mergedConfig = dcm.mergeConfigs(mergedConfig, config)
        }
    }

    if mergedConfig == nil {
        return fmt.Errorf("no valid config sources available")
    }

    return dcm.UpdateConfig(mergedConfig)
}

func (dcm *DefaultConfigurationManager) AddConfigSource(source ConfigSource) error {
    dcm.mutex.Lock()
    dcm.configSources = append(dcm.configSources, source)
    dcm.mutex.Unlock()

    // 启动配置监听
    if source.SupportsReload() {
        return source.WatchChanges(func(config *Config) {
            if err := dcm.ReloadConfig(); err != nil {
                log.Printf("Failed to reload config: %v", err)
            }
        })
    }

    return nil
}

func (dcm *DefaultConfigurationManager) WatchConfig(callback ConfigChangeCallback) error {
    dcm.mutex.Lock()
    defer dcm.mutex.Unlock()

    dcm.callbacks = append(dcm.callbacks, callback)
    return nil
}

func (dcm *DefaultConfigurationManager) RollbackTo(version string) error {
    dcm.mutex.Lock()
    defer dcm.mutex.Unlock()

    // 查找指定版本的配置
    for _, snapshot := range dcm.history {
        if snapshot.Version == version {
            return dcm.UpdateConfig(snapshot.Config)
        }
    }

    return fmt.Errorf("config version not found: %s", version)
}
```

### 4. 配置验证器

```go
// 配置验证器接口
type ConfigValidator interface {
    Validate(config *Config) []ConfigError
    ValidateChange(oldConfig, newConfig *Config) []ConfigError
}

type ConfigError struct {
    Path    string `json:"path"`
    Field   string `json:"field"`
    Message string `json:"message"`
    Code    string `json:"code"`
}

// 默认配置验证器
type DefaultConfigValidator struct {
    platformValidators map[string]PlatformValidator
}

type PlatformValidator interface {
    ValidatePlatformConfig(config PlatformConfig) []ConfigError
}

func (dcv *DefaultConfigValidator) Validate(config *Config) []ConfigError {
    var errors []ConfigError

    // 验证基础配置
    errors = append(errors, dcv.validateBasicConfig(config)...)

    // 验证平台配置
    for _, platform := range config.Platforms {
        errors = append(errors, dcv.validatePlatformConfig(platform)...)
    }

    // 验证队列配置
    if config.Queue != nil {
        errors = append(errors, dcv.validateQueueConfig(config.Queue)...)
    }

    // 验证重试配置
    if config.Retry != nil {
        errors = append(errors, dcv.validateRetryConfig(config.Retry)...)
    }

    return errors
}

func (dcv *DefaultConfigValidator) validatePlatformConfig(platform PlatformConfig) []ConfigError {
    var errors []ConfigError

    if platform.Name == "" {
        errors = append(errors, ConfigError{
            Path:    "platforms",
            Field:   "name",
            Message: "platform name is required",
            Code:    "REQUIRED_FIELD",
        })
    }

    if platform.Type == "" {
        errors = append(errors, ConfigError{
            Path:    "platforms",
            Field:   "type",
            Message: "platform type is required",
            Code:    "REQUIRED_FIELD",
        })
    }

    // 平台特定验证
    if validator, exists := dcv.platformValidators[string(platform.Type)]; exists {
        errors = append(errors, validator.ValidatePlatformConfig(platform)...)
    }

    return errors
}

// 飞书平台验证器
type FeishuValidator struct{}

func (fv *FeishuValidator) ValidatePlatformConfig(config PlatformConfig) []ConfigError {
    var errors []ConfigError

    webhook, ok := config.Settings["webhook"].(string)
    if !ok || webhook == "" {
        errors = append(errors, ConfigError{
            Path:    "platforms.feishu.settings",
            Field:   "webhook",
            Message: "webhook URL is required for Feishu platform",
            Code:    "REQUIRED_FIELD",
        })
    } else if !isValidURL(webhook) {
        errors = append(errors, ConfigError{
            Path:    "platforms.feishu.settings",
            Field:   "webhook",
            Message: "invalid webhook URL format",
            Code:    "INVALID_FORMAT",
        })
    }

    return errors
}
```

### 5. 热重载触发器

```go
// HTTP配置API
type ConfigurationAPI struct {
    manager    ConfigurationManager
    auth       AuthService
    validator  RequestValidator
}

func (ca *ConfigurationAPI) setupRoutes() *http.ServeMux {
    mux := http.NewServeMux()

    mux.HandleFunc("/config", ca.authMiddleware(ca.handleConfig))
    mux.HandleFunc("/config/reload", ca.authMiddleware(ca.handleReload))
    mux.HandleFunc("/config/validate", ca.authMiddleware(ca.handleValidate))
    mux.HandleFunc("/config/history", ca.authMiddleware(ca.handleHistory))
    mux.HandleFunc("/config/rollback", ca.authMiddleware(ca.handleRollback))
    mux.HandleFunc("/config/diff", ca.authMiddleware(ca.handleDiff))

    return mux
}

func (ca *ConfigurationAPI) handleConfig(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
    case http.MethodGet:
        config := ca.manager.GetCurrentConfig()
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(config)

    case http.MethodPut:
        var newConfig Config
        if err := json.NewDecoder(r.Body).Decode(&newConfig); err != nil {
            http.Error(w, "Invalid JSON", http.StatusBadRequest)
            return
        }

        if err := ca.manager.UpdateConfig(&newConfig); err != nil {
            http.Error(w, err.Error(), http.StatusBadRequest)
            return
        }

        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(map[string]string{
            "status": "updated",
        })

    default:
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
    }
}

func (ca *ConfigurationAPI) handleReload(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    if err := ca.manager.ReloadConfig(); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{
        "status": "reloaded",
    })
}

func (ca *ConfigurationAPI) handleValidate(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    var config Config
    if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
        http.Error(w, "Invalid JSON", http.StatusBadRequest)
        return
    }

    errors := ca.manager.ValidateConfig(&config)

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "valid":  len(errors) == 0,
        "errors": errors,
    })
}

func (ca *ConfigurationAPI) handleRollback(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    version := r.URL.Query().Get("version")
    if version == "" {
        http.Error(w, "version parameter is required", http.StatusBadRequest)
        return
    }

    if err := ca.manager.RollbackTo(version); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{
        "status": "rolled back",
        "version": version,
    })
}
```

## 集成和使用

### 1. 客户端配置

```go
// 配置热重载选项
func WithHotReload(config HotReloadConfig) Option {
    return func(cfg *Config) {
        cfg.HotReload = &config
    }
}

type HotReloadConfig struct {
    Enabled      bool            `json:"enabled"`
    Sources      []ConfigSource  `json:"sources"`
    APIEnabled   bool           `json:"api_enabled"`
    APIPort      int            `json:"api_port"`
    APIAuth      AuthConfig     `json:"api_auth"`
    ValidateOnly bool           `json:"validate_only"` // 仅验证不应用
}

// 使用示例
client, err := notifyhub.New(
    notifyhub.WithFeishu("webhook", "secret"),
    notifyhub.WithHotReload(HotReloadConfig{
        Enabled:    true,
        APIEnabled: true,
        APIPort:    8081,
        Sources: []ConfigSource{
            NewFileConfigSource("/etc/notifyhub/config.yaml", 100),
            NewConsulConfigSource(consulClient, "notifyhub/config", 90),
            NewHTTPConfigSource("https://config.company.com/notifyhub", 80),
        },
        APIAuth: AuthConfig{
            Type: "token",
            Token: "admin-token-123",
        },
    }),
)

// 监听配置变更
client.WatchConfig(func(oldConfig, newConfig *Config, diff ConfigDiff) error {
    log.Printf("Config changed: %+v", diff)

    // 可以在这里添加自定义逻辑
    if len(diff.Modified) > 0 {
        log.Printf("Modified fields: %v", diff.Modified)
    }

    return nil
})
```

### 2. 配置文件示例

```yaml
# /etc/notifyhub/config.yaml
platforms:
  - type: feishu
    name: feishu-main
    enabled: true
    settings:
      webhook: "https://open.feishu.cn/webhook/xxx"
      secret: "secret123"

  - type: email
    name: email-main
    enabled: true
    settings:
      host: "smtp.company.com"
      port: 587
      username: "noreply@company.com"
      password: "password123"
      from: "NotifyHub <noreply@company.com>"

queue:
  type: "redis"
  capacity: 5000
  concurrency: 8

retry:
  max_attempts: 5
  backoff: "2s"
  jitter: true

rate_limit:
  rate: 100
  burst: 200
  window: "1m"
```

### 3. API使用示例

```bash
# 获取当前配置
curl -H "Authorization: Bearer admin-token-123" \
     http://localhost:8081/config

# 更新配置
curl -X PUT \
     -H "Authorization: Bearer admin-token-123" \
     -H "Content-Type: application/json" \
     -d '{"platforms":[...]}' \
     http://localhost:8081/config

# 重载配置
curl -X POST \
     -H "Authorization: Bearer admin-token-123" \
     http://localhost:8081/config/reload

# 验证配置
curl -X POST \
     -H "Authorization: Bearer admin-token-123" \
     -H "Content-Type: application/json" \
     -d '{"platforms":[...]}' \
     http://localhost:8081/config/validate

# 回滚配置
curl -X POST \
     -H "Authorization: Bearer admin-token-123" \
     http://localhost:8081/config/rollback?version=v1.2.3

# 查看配置历史
curl -H "Authorization: Bearer admin-token-123" \
     http://localhost:8081/config/history

# 配置差异对比
curl -X POST \
     -H "Authorization: Bearer admin-token-123" \
     -H "Content-Type: application/json" \
     -d '{"old_version":"v1.2.3","new_version":"v1.2.4"}' \
     http://localhost:8081/config/diff
```

## 安全考虑

### 1. 访问控制

```go
// 认证服务
type AuthService interface {
    ValidateToken(token string) (*AuthContext, error)
    HasPermission(ctx *AuthContext, action string) bool
}

type AuthContext struct {
    UserID      string   `json:"user_id"`
    Username    string   `json:"username"`
    Roles       []string `json:"roles"`
    Permissions []string `json:"permissions"`
}

// 权限定义
const (
    PermissionConfigRead   = "config:read"
    PermissionConfigWrite  = "config:write"
    PermissionConfigReload = "config:reload"
    PermissionConfigRollback = "config:rollback"
)
```

### 2. 配置加密

```go
// 敏感配置加密
type ConfigEncryption interface {
    Encrypt(plaintext string) (string, error)
    Decrypt(ciphertext string) (string, error)
    EncryptConfig(config *Config) (*Config, error)
    DecryptConfig(config *Config) (*Config, error)
}

// 使用示例
encryptedConfig, err := encryption.EncryptConfig(config)
```

## 总结

配置热重载方案提供了：

1. **多源配置支持**：文件、HTTP、Consul等多种配置源
2. **实时配置监听**：自动检测配置变更并应用
3. **配置验证机制**：确保配置正确性和兼容性
4. **版本管理**：配置历史记录和回滚功能
5. **RESTful API**：便于集成和自动化
6. **安全保障**：完善的认证和权限控制

该方案极大提升了NotifyHub在生产环境中的运维便利性，支持零停机的配置调整和优化。