package config

import (
	"os"
	"strings"
	"testing"
)

// createTempConfigFile 是一个辅助函数，用于创建一个包含指定内容的临时 YAML 配置文件。
// 它会返回创建的文件的路径，并在测试结束后自动清理该文件。
func createTempConfigFile(t *testing.T, content string) string {
	t.Helper()
	// 创建一个临时文件
	tmpfile, err := os.CreateTemp("", "test_config_*.yml")
	if err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}

	// 将内容写入文件
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatalf("写入临时文件失败: %v", err)
	}

	// 关闭文件
	if err := tmpfile.Close(); err != nil {
		t.Fatalf("关闭临时文件失败: %v", err)
	}

	// 注册一个清理函数，在测试结束后删除临时文件
	t.Cleanup(func() {
		os.Remove(tmpfile.Name())
	})

	return tmpfile.Name()
}

func TestLoadSeatConfig(t *testing.T) {
	// 定义一个基础的、有效的 YAML 配置字符串，后续的测试用例将在此基础上进行修改。
	// --- 修正：将全角冒号 '：' 改为半角冒号 ':' ---
	baseValidYAML := `
global:
  preempt_seconds: 15
week_config:
  周一:
    启用: true
    run_at_hour: 20
    run_at_minute: 30
    name: "测试自习室"
    seats: ["101", "102"]
    book_start_hour: 8
    duration: 10
`

	// 定义一系列测试用例
	testCases := []struct {
		name        string              // 测试用例的名称
		modifier    func(string) string // 一个函数，用于修改基础 YAML 以产生测试所需的内容
		expectErr   bool                // 是否期望出现错误
		errContains string              // 如果期望出错，错误信息中应包含的子字符串
	}{
		{
			name:      "有效配置",
			modifier:  func(y string) string { return y }, // 不做任何修改
			expectErr: false,
		},
		{
			name: "无效的 book_start_hour (过早)",
			modifier: func(y string) string {
				return strings.Replace(y, "book_start_hour: 8", "book_start_hour: 6", 1)
			},
			expectErr:   true,
			errContains: "BookStartHour", // 对应你的错误信息
		},
		{
			name: "无效的 book_start_hour (过晚)",
			modifier: func(y string) string {
				return strings.Replace(y, "book_start_hour: 8", "book_start_hour: 23", 1)
			},
			expectErr:   true,
			errContains: "BookStartHour", // 对应你的错误信息
		},
		{
			name: "无效的 run_at_hour",
			modifier: func(y string) string {
				return strings.Replace(y, "run_at_hour: 20", "run_at_hour: 25", 1)
			},
			expectErr:   true,
			errContains: "Run_At_Hour", // 对应你的错误信息
		},
		{
			name: "无效的 run_at_minute",
			modifier: func(y string) string {
				return strings.Replace(y, "run_at_minute: 30", "run_at_minute: 61", 1)
			},
			expectErr:   true,
			errContains: "Run_At_Minute", // 对应你的错误信息
		},
		{
			name: "无效的持续时间 (开始时间 + 持续时间 > 22)",
			modifier: func(y string) string {
				// book_start_hour: 8, duration: 15  => 8 + 15 = 23, 超过了 22
				return strings.Replace(y, "duration: 10", "duration: 15", 1)
			},
			expectErr:   true,
			errContains: "Duration+BookStartHour", // 对应你的错误信息
		},
	}

	// 遍历并执行所有测试用例
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 1. 根据测试用例生成 YAML 内容并创建临时文件
			yamlContent := tc.modifier(baseValidYAML)
			filePath := createTempConfigFile(t, yamlContent)

			// 2. 调用被测试的函数
			_, err := LoadSeatConfig(filePath)

			// 3. 断言结果
			if tc.expectErr {
				// 如果期望出错
				if err == nil {
					t.Errorf("期望出现错误，但返回的错误为 nil")
				} else if tc.errContains != "" && !strings.Contains(err.Error(), tc.errContains) {
					// 检查错误信息是否符合预期
					t.Errorf("期望错误信息包含 '%s', 但实际错误是: %v", tc.errContains, err)
				}
			} else {
				// 如果不期望出错
				if err != nil {
					t.Errorf("不期望出现错误，但收到了错误: %v", err)
				}
			}
		})
	}
}
