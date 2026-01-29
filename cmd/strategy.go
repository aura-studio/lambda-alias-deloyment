// Package cmd implements the command line interface for the lad tool.
package cmd

// CanaryStrategy 灰度策略
type CanaryStrategy string

const (
	// Canary0 0% 灰度策略（清除灰度，回到旧版本）
	Canary0 CanaryStrategy = "canary0"
	// Canary10 10% 灰度策略
	Canary10 CanaryStrategy = "canary10"
	// Canary25 25% 灰度策略
	Canary25 CanaryStrategy = "canary25"
	// Canary50 50% 灰度策略
	Canary50 CanaryStrategy = "canary50"
	// Canary75 75% 灰度策略
	Canary75 CanaryStrategy = "canary75"
	// Canary100 100% 灰度策略（全量切换，但不更新 previous）
	Canary100 CanaryStrategy = "canary100"
)

// AllStrategies 返回所有有效的灰度策略列表
var AllStrategies = []CanaryStrategy{Canary0, Canary10, Canary25, Canary50, Canary75, Canary100}

// Weight 返回策略对应的权重
func (s CanaryStrategy) Weight() float64 {
	switch s {
	case Canary0:
		return 0.0
	case Canary10:
		return 0.10
	case Canary25:
		return 0.25
	case Canary50:
		return 0.50
	case Canary75:
		return 0.75
	case Canary100:
		return 1.0
	default:
		return 0
	}
}

// IsValid 验证策略是否有效
func (s CanaryStrategy) IsValid() bool {
	switch s {
	case Canary0, Canary10, Canary25, Canary50, Canary75, Canary100:
		return true
	default:
		return false
	}
}

// NextStrategy 返回下一个策略
// 策略顺序: canary0 -> canary10 -> canary25 -> canary50 -> canary75 -> canary100 -> canary100
func (s CanaryStrategy) NextStrategy() CanaryStrategy {
	switch s {
	case Canary0:
		return Canary10
	case Canary10:
		return Canary25
	case Canary25:
		return Canary50
	case Canary50:
		return Canary75
	case Canary75:
		return Canary100
	case Canary100:
		return Canary100 // 最后一个策略返回自身
	default:
		return Canary10 // 无效策略返回第一个策略
	}
}
