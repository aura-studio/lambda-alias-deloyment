// Package output provides formatted output utilities for the lad command line tool.
package output

import (
	"fmt"
	"os"
)

// Info 输出普通信息到 stdout
func Info(format string, args ...interface{}) {
	fmt.Fprintf(os.Stdout, format+"\n", args...)
}

// Error 输出错误信息到 stderr，格式为"错误: {message}"
func Error(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stderr, "错误: %s\n", message)
}

// Success 输出成功标记 "✓ {message}"
func Success(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stdout, "✓ %s\n", message)
}

// Warning 输出警告信息 "⚠ {message}"
func Warning(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stdout, "⚠ %s\n", message)
}

// Separator 输出分隔线 "=========================================="
func Separator() {
	fmt.Fprintln(os.Stdout, "==========================================")
}
