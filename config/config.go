package config

import (
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
	return &config, nil
}
