package queue

import (
	"fmt"
	"sync"
)

// ExternalRegistry 外部队列工厂注册表
type ExternalRegistry struct {
	mu        sync.RWMutex
	factories map[string]ExternalQueueFactory
}

// NewExternalRegistry 创建新的外部注册表
func NewExternalRegistry() *ExternalRegistry {
	return &ExternalRegistry{
		factories: make(map[string]ExternalQueueFactory),
	}
}

// Register 注册队列工厂
func (r *ExternalRegistry) Register(factory ExternalQueueFactory) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := factory.Name()
	if name == "" {
		return fmt.Errorf("factory name cannot be empty")
	}

	if _, exists := r.factories[name]; exists {
		return fmt.Errorf("factory '%s' already registered", name)
	}

	r.factories[name] = factory
	return nil
}

// Unregister 注销队列工厂
func (r *ExternalRegistry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.factories, name)
}

// Get 获取队列工厂
func (r *ExternalRegistry) Get(name string) (ExternalQueueFactory, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	factory, exists := r.factories[name]
	if !exists {
		return nil, fmt.Errorf("factory '%s' not found", name)
	}

	return factory, nil
}

// Create 创建队列实例
func (r *ExternalRegistry) Create(queueType string, config map[string]interface{}) (ExternalQueue, error) {
	factory, err := r.Get(queueType)
	if err != nil {
		return nil, err
	}

	// 验证配置
	if err := factory.Validate(config); err != nil {
		return nil, fmt.Errorf("invalid config for '%s': %v", queueType, err)
	}

	return factory.Create(config)
}

// List 列出所有已注册的队列类型
func (r *ExternalRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.factories))
	for name := range r.factories {
		names = append(names, name)
	}
	return names
}

// 全局默认外部注册表
var defaultExternalRegistry = NewExternalRegistry()

// RegisterExternal 注册到默认外部注册表
func RegisterExternal(factory ExternalQueueFactory) error {
	return defaultExternalRegistry.Register(factory)
}

// UnregisterExternal 从默认外部注册表注销
func UnregisterExternal(name string) {
	defaultExternalRegistry.Unregister(name)
}

// GetExternal 从默认外部注册表获取工厂
func GetExternal(name string) (ExternalQueueFactory, error) {
	return defaultExternalRegistry.Get(name)
}

// CreateExternal 使用默认外部注册表创建队列
func CreateExternal(queueType string, config map[string]interface{}) (ExternalQueue, error) {
	return defaultExternalRegistry.Create(queueType, config)
}

// ListExternal 列出默认外部注册表中的所有队列类型
func ListExternal() []string {
	return defaultExternalRegistry.List()
}
