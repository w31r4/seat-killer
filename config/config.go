package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// --- UserInfo ---
type UserInfo struct {
	SchoolID string `yaml:"school_id"`
	Password string `yaml:"password"`
}

func LoadUserInfo(path string) (*UserInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var userInfo UserInfo
	err = yaml.Unmarshal(data, &userInfo)
	if err != nil {
		return nil, err
	}
	return &userInfo, nil
}

// --- SeatConfig (Final Structure) ---

// SeatConfig is the main configuration structure.
type SeatConfig struct {
	Global     GlobalConfig         `yaml:"global"`
	WeekConfig map[string]DayConfig `yaml:"week_config"`
}

// GlobalConfig holds settings that apply to all tasks.
// 全局配置，目前只有提前开始时间一个配置变量
type GlobalConfig struct {
	PreemptSeconds int `yaml:"preempt_seconds"`
}

// DayConfig represents the configuration for a specific day of the week.
// It contains a single task with a prioritized list of seats.
type DayConfig struct {
	Enable        bool     `yaml:"启用"`
	RunAtHour     int      `yaml:"run_at_hour"`
	RunAtMinute   int      `yaml:"run_at_minute"` // Temporary for testing
	Name          string   `yaml:"name"`
	Seats         []string `yaml:"seats"`
	BookStartHour int      `yaml:"book_start_hour"`
	Duration      int      `yaml:"duration"`
}

func LoadSeatConfig(path string) (*SeatConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var config SeatConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}
	for day, dayConfig := range config.WeekConfig {
		if dayConfig.RunAtHour > 24 || dayConfig.RunAtHour < 0 {
			return nil, fmt.Errorf("配置校验失败->%s的'Run_At_Hour'(%d)无效,必须在0-24之间'", day, dayConfig.RunAtHour)
		}
		if dayConfig.RunAtMinute > 60 || dayConfig.RunAtMinute < 0 {
			return nil, fmt.Errorf("配置校验失败->%s的'Run_At_Minute'(%d)无效,必须在0-60之间'", day, dayConfig.RunAtMinute)
		}
		if dayConfig.BookStartHour < 7 || dayConfig.BookStartHour > 22 {
			return nil, fmt.Errorf("配置校验失败->%s的'BookStartHour'(%d)无效,必须在7-22之间'", day, dayConfig.BookStartHour)
		}
		if dayConfig.BookStartHour+dayConfig.Duration > 22 {
			return nil, fmt.Errorf("配置校验失败->%s的'Duration+BookStartHour'(%d)超出合理范围,结果必须在7-22之间'", day, dayConfig.BookStartHour+dayConfig.Duration)
		}

	}
	return &config, nil
}
