// Package cmd implements the command line interface for the lad tool.
package cmd

import "fmt"

// CanaryStrategy 灰度策略
type CanaryStrategy string

const (
	// Canary10 10% 灰度策略
	Canary10 CanaryStrategy = "canary10"
	// Canary25 25% 灰度策略
	Canary25 CanaryStrategy = "canary25"
	// Canary50 50% 灰度策略
	Canary50 CanaryStrategy = "canary50"
	// Canary75 75% 灰度策略
	Canary75 CanaryStrategy = "canary75"
)

// AllStrategies 返回所有有效的灰度策略列表
var AllStrategies = []CanaryStrategy{Canary10, Canary25, Canary50, Canary75}

// Weight 返回策略对应的权重
func (s CanaryStrategy) Weight() float64 {
	switch s {
	case Canary10:
		return 0.10
	case Canary25:
		return 0.25
	case Canary50:
		return 0.50
	case Canary75:
		return 0.75
	default:
		return 0
	}
}

// IsValid 验证策略是否有效
func (s CanaryStrategy) IsValid() bool {
	switch s {
	case Canary10, Canary25, Canary50, Canary75:
		return true
	default:
		return false
	}
}

// NextStrategy 返回下一个策略
// 策略顺序: canary10 -> canary25 -> canary50 -> canary75 -> canary75 (最后一个返回自身)
func (s CanaryStrategy) NextStrategy() CanaryStrategy {
	switch s {
	case Canary10:
		return Canary25
	case Canary25:
		return Canary50
	case Canary50:
		return Canary75
	case Canary75:
		return Canary75 // 最后一个策略返回自身
	default:
		return Canary10 // 无效策略返回第一个策略
	}
}

// ErrAutoPromoteOnlyCanary75 is the error returned when --auto-promote is used with a non-canary75 strategy
var ErrAutoPromoteOnlyCanary75 = fmt.Errorf("--auto-promote 仅在策略为 canary75 时可用")

// ValidateAutoPromote validates the --auto-promote parameter
// Returns an error if autoPromote is true but strategy is not Canary75
// **Validates: Requirements 5.8**
func ValidateAutoPromote(strategy CanaryStrategy, autoPromote bool) error {
	if autoPromote && strategy != Canary75 {
		return ErrAutoPromoteOnlyCanary75
	}
	return nil
}
