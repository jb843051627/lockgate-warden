// Package config 从环境变量装配服务参数。
package config

import (
	"os"
	"strconv"
	"time"
)

// Config 汇总进程启动所需全部参数。
type Config struct {
	Addr        string
	DBPath      string
	DedupWindow time.Duration
	Staleness   time.Duration
	WarnTTL     time.Duration
	Retention   time.Duration
	MaxSpan     time.Duration
	WireWindow  time.Duration // 闸位计滑动窗口
	QueueBuffer int
}

// FromEnv 读取环境变量并应用默认值。
func FromEnv() Config {
	return Config{
		Addr:        env("LOCKGATE_ADDR", ":8080"),
		DBPath:      env("LOCKGATE_DB", "lockgate.db"),
		DedupWindow: envDur("LOCKGATE_DEDUP", 30*time.Minute),
		Staleness:   envDur("LOCKGATE_STALENESS", 10*time.Minute),
		WarnTTL:     envDur("LOCKGATE_WARN_TTL", 6*time.Hour),
		Retention:   envDur("LOCKGATE_RETENTION", 48*time.Hour),
		MaxSpan:     envDur("LOCKGATE_MAX_SPAN", 6*time.Hour),
		WireWindow:  envDur("LOCKGATE_WIRE_WINDOW", 24*time.Hour),
		QueueBuffer: envInt("LOCKGATE_QUEUE_BUFFER", 64),
	}
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envDur(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}

func envInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}
