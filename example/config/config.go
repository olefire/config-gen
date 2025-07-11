// Code generated by config-gen; DO NOT EDIT.
package config

import (
	"context"
	"time"

	konfig "github.com/olefire/realtime-config-go"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type AppConfig interface {
	// GetEnableFeature возвращает значение enable_feature. feature flag
	GetEnableFeature() bool
	// GetThresholds возвращает значение thresholds. Пороговые значения
	GetThresholds() map[string]int
	// GetTimeout возвращает значение timeout. Таймаут ожидания ответа от сервиса
	GetTimeout() time.Duration
	// GetAuthRequiredMethods возвращает значение auth_required_methods. Методы, требующие аутентификации
	GetAuthRequiredMethods() map[string]struct{}
	// GetAppName возвращает значение app_name. Название приложения
	GetAppName() string
	// GetPort возвращает значение port. Порт сервера
	GetPort() int
}

type appConfig struct {
	enableFeature       bool                `etcd:"enable_feature"`
	thresholds          map[string]int      `etcd:"thresholds"`
	timeout             time.Duration       `etcd:"timeout"`
	authRequiredMethods map[string]struct{} `etcd:"auth_required_methods"`
	appName             string              `etcd:"app_name"`
	port                int                 `etcd:"port"`
}

func NewAppConfig(ctx context.Context, cli *clientv3.Client, prefix string) (*konfig.RealTimeConfig, error) {
	cfg := &appConfig{
		enableFeature: true,
		thresholds:    map[string]int{"warning": 70, "critical": 90},
		timeout: func() time.Duration {
			d, _ := time.ParseDuration("5s")
			return d
		}(),
		authRequiredMethods: map[string]struct{}{"/api/user/profile": struct{}{}, "/api/user/settings": struct{}{}, "/api/orders/create": struct{}{}},
		appName:             "myapp",
		port:                8080,
	}
	return konfig.NewRealTimeConfig(ctx, cli, prefix, cfg)
}

// GetEnableFeature возвращает значение enable_feature. feature flag
func (c *appConfig) GetEnableFeature() bool {
	return c.enableFeature
}

// GetThresholds возвращает значение thresholds. Пороговые значения
func (c *appConfig) GetThresholds() map[string]int {
	return c.thresholds
}

// GetTimeout возвращает значение timeout. Таймаут ожидания ответа от сервиса
func (c *appConfig) GetTimeout() time.Duration {
	return c.timeout
}

// GetAuthRequiredMethods возвращает значение auth_required_methods. Методы, требующие аутентификации
func (c *appConfig) GetAuthRequiredMethods() map[string]struct{} {
	return c.authRequiredMethods
}

// GetAppName возвращает значение app_name. Название приложения
func (c *appConfig) GetAppName() string {
	return c.appName
}

// GetPort возвращает значение port. Порт сервера
func (c *appConfig) GetPort() int {
	return c.port
}
