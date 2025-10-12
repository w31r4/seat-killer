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

// --- SeatConfig ---
type SeatConfig struct {
	WeekConfig map[string]DayConfig `yaml:",inline"`
}

type DayConfig struct {
	Enable        bool   `yaml:"启用"`
	Name          string `yaml:"name"`
	Seat          string `yaml:"seat"`
	RunAtHour     int    `yaml:"run_at_hour"`     // The hour the script should start trying to book (e.g., 20 for 8 PM)
	BookStartHour int    `yaml:"book_start_hour"` // The actual start hour of the seat booking (e.g., 7 for 7 AM)
	Duration      int    `yaml:"duration"`
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
