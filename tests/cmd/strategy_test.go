package cmd_test

import (
	"testing"

	"github.com/aura-studio/lambda-alias-deployment/cmd"
	"pgregory.net/rapid"
)

func TestCanaryStrategy_Weight(t *testing.T) {
	tests := []struct {
		name     string
		strategy cmd.CanaryStrategy
		want     float64
	}{
		{
			name:     "canary10 returns 0.10",
			strategy: cmd.Canary10,
			want:     0.10,
		},
		{
			name:     "canary25 returns 0.25",
			strategy: cmd.Canary25,
			want:     0.25,
		},
		{
			name:     "canary50 returns 0.50",
			strategy: cmd.Canary50,
			want:     0.50,
		},
		{
			name:     "canary75 returns 0.75",
			strategy: cmd.Canary75,
			want:     0.75,
		},
		{
			name:     "invalid strategy returns 0",
			strategy: cmd.CanaryStrategy("invalid"),
			want:     0,
		},
		{
			name:     "empty strategy returns 0",
			strategy: cmd.CanaryStrategy(""),
			want:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.strategy.Weight()
			if got != tt.want {
				t.Errorf("CanaryStrategy.Weight() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCanaryStrategy_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		strategy cmd.CanaryStrategy
		want     bool
	}{
		{
			name:     "canary10 is valid",
			strategy: cmd.Canary10,
			want:     true,
		},
		{
			name:     "canary25 is valid",
			strategy: cmd.Canary25,
			want:     true,
		},
		{
			name:     "canary50 is valid",
			strategy: cmd.Canary50,
			want:     true,
		},
		{
			name:     "canary75 is valid",
			strategy: cmd.Canary75,
			want:     true,
		},
		{
			name:     "invalid strategy is not valid",
			strategy: cmd.CanaryStrategy("invalid"),
			want:     false,
		},
		{
			name:     "empty strategy is not valid",
			strategy: cmd.CanaryStrategy(""),
			want:     false,
		},
		{
			name:     "canary100 is not valid",
			strategy: cmd.CanaryStrategy("canary100"),
			want:     false,
		},
		{
			name:     "CANARY10 (uppercase) is not valid",
			strategy: cmd.CanaryStrategy("CANARY10"),
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.strategy.IsValid()
			if got != tt.want {
				t.Errorf("CanaryStrategy.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCanaryStrategy_NextStrategy(t *testing.T) {
	tests := []struct {
		name     string
		strategy cmd.CanaryStrategy
		want     cmd.CanaryStrategy
	}{
		{
			name:     "canary10 -> canary25",
			strategy: cmd.Canary10,
			want:     cmd.Canary25,
		},
		{
			name:     "canary25 -> canary50",
			strategy: cmd.Canary25,
			want:     cmd.Canary50,
		},
		{
			name:     "canary50 -> canary75",
			strategy: cmd.Canary50,
			want:     cmd.Canary75,
		},
		{
			name:     "canary75 -> canary75 (stays at max)",
			strategy: cmd.Canary75,
			want:     cmd.Canary75,
		},
		{
			name:     "invalid strategy -> canary10",
			strategy: cmd.CanaryStrategy("invalid"),
			want:     cmd.Canary10,
		},
		{
			name:     "empty strategy -> canary10",
			strategy: cmd.CanaryStrategy(""),
			want:     cmd.Canary10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.strategy.NextStrategy()
			if got != tt.want {
				t.Errorf("CanaryStrategy.NextStrategy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAllStrategies(t *testing.T) {
	// Verify AllStrategies contains all valid strategies
	expected := []cmd.CanaryStrategy{cmd.Canary10, cmd.Canary25, cmd.Canary50, cmd.Canary75}

	if len(cmd.AllStrategies) != len(expected) {
		t.Errorf("AllStrategies has %d elements, want %d", len(cmd.AllStrategies), len(expected))
	}

	for i, s := range expected {
		if cmd.AllStrategies[i] != s {
			t.Errorf("AllStrategies[%d] = %v, want %v", i, cmd.AllStrategies[i], s)
		}
	}

	// Verify all strategies in AllStrategies are valid
	for _, s := range cmd.AllStrategies {
		if !s.IsValid() {
			t.Errorf("Strategy %v in AllStrategies is not valid", s)
		}
	}
}

func TestCanaryStrategy_WeightConsistency(t *testing.T) {
	// Verify that weights are in ascending order
	strategies := []cmd.CanaryStrategy{cmd.Canary10, cmd.Canary25, cmd.Canary50, cmd.Canary75}
	expectedWeights := []float64{0.10, 0.25, 0.50, 0.75}

	for i, s := range strategies {
		if s.Weight() != expectedWeights[i] {
			t.Errorf("Strategy %v has weight %v, want %v", s, s.Weight(), expectedWeights[i])
		}
	}

	// Verify weights are in ascending order
	for i := 1; i < len(strategies); i++ {
		if strategies[i].Weight() <= strategies[i-1].Weight() {
			t.Errorf("Weight of %v (%v) should be greater than %v (%v)",
				strategies[i], strategies[i].Weight(),
				strategies[i-1], strategies[i-1].Weight())
		}
	}
}

// =============================================================================
// Property-Based Tests
// =============================================================================

// TestProperty3_CanaryStrategyValidation tests Property 3: 灰度策略验证
// **Validates: Requirements 5.2, 5.3**
//
// Property 3: For any canary strategy string, if the value is one of
// canary10/canary25/canary50/canary75, it should return the corresponding
// weight (0.10/0.25/0.50/0.75); otherwise it should return 0 (invalid).
func TestProperty3_CanaryStrategyValidation(t *testing.T) {
	// Define the valid strategies and their expected weights
	validStrategies := map[string]float64{
		"canary10": 0.10,
		"canary25": 0.25,
		"canary50": 0.50,
		"canary75": 0.75,
	}

	// Property 3a: Valid strategies return correct weights
	t.Run("valid_strategies_return_correct_weights", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			// Generate a random valid strategy from the list
			strategyIndex := rapid.IntRange(0, len(cmd.AllStrategies)-1).Draw(t, "strategyIndex")
			strategy := cmd.AllStrategies[strategyIndex]

			// Verify the strategy is valid
			if !strategy.IsValid() {
				t.Fatalf("Strategy %q from AllStrategies should be valid", strategy)
			}

			// Verify the weight matches expected value
			expectedWeight := validStrategies[string(strategy)]
			actualWeight := strategy.Weight()
			if actualWeight != expectedWeight {
				t.Fatalf("Strategy %q: expected weight %v, got %v", strategy, expectedWeight, actualWeight)
			}
		})
	})

	// Property 3b: Invalid strategies return 0 weight and IsValid returns false
	t.Run("invalid_strategies_return_zero_weight", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			// Generate a random string that is NOT a valid strategy
			randomStr := rapid.StringMatching(`[a-zA-Z0-9_-]{0,20}`).Draw(t, "randomStr")
			strategy := cmd.CanaryStrategy(randomStr)

			// Skip if we accidentally generated a valid strategy
			if _, isValid := validStrategies[randomStr]; isValid {
				return
			}

			// Verify the strategy is invalid
			if strategy.IsValid() {
				t.Fatalf("Strategy %q should be invalid", strategy)
			}

			// Verify the weight is 0 for invalid strategies
			if strategy.Weight() != 0 {
				t.Fatalf("Invalid strategy %q should return weight 0, got %v", strategy, strategy.Weight())
			}
		})
	})

	// Property 3c: All valid strategy strings map to exactly one weight
	t.Run("valid_strategy_weight_mapping_is_bijective", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			// Generate two random valid strategies
			idx1 := rapid.IntRange(0, len(cmd.AllStrategies)-1).Draw(t, "idx1")
			idx2 := rapid.IntRange(0, len(cmd.AllStrategies)-1).Draw(t, "idx2")

			s1 := cmd.AllStrategies[idx1]
			s2 := cmd.AllStrategies[idx2]

			// If strategies are different, their weights should be different
			if s1 != s2 && s1.Weight() == s2.Weight() {
				t.Fatalf("Different strategies %q and %q should have different weights, both have %v",
					s1, s2, s1.Weight())
			}

			// If strategies are the same, their weights should be the same
			if s1 == s2 && s1.Weight() != s2.Weight() {
				t.Fatalf("Same strategy %q should have consistent weight", s1)
			}
		})
	})

	// Property 3d: Weight values are within valid range (0.0 to 1.0)
	t.Run("weights_are_within_valid_range", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			// Generate a random valid strategy
			strategyIndex := rapid.IntRange(0, len(cmd.AllStrategies)-1).Draw(t, "strategyIndex")
			strategy := cmd.AllStrategies[strategyIndex]

			weight := strategy.Weight()

			// Weight should be > 0 for valid strategies
			if weight <= 0 {
				t.Fatalf("Valid strategy %q should have weight > 0, got %v", strategy, weight)
			}

			// Weight should be < 1.0 (we don't have 100% canary)
			if weight >= 1.0 {
				t.Fatalf("Valid strategy %q should have weight < 1.0, got %v", strategy, weight)
			}
		})
	})

	// Property 3e: AllStrategies contains exactly the valid strategies
	t.Run("all_strategies_list_is_complete", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			// Generate a random string
			randomStr := rapid.String().Draw(t, "randomStr")
			strategy := cmd.CanaryStrategy(randomStr)

			// Check if it's in AllStrategies
			inAllStrategies := false
			for _, s := range cmd.AllStrategies {
				if s == strategy {
					inAllStrategies = true
					break
				}
			}

			// If it's valid, it should be in AllStrategies
			if strategy.IsValid() && !inAllStrategies {
				t.Fatalf("Valid strategy %q should be in AllStrategies", strategy)
			}

			// If it's in AllStrategies, it should be valid
			if inAllStrategies && !strategy.IsValid() {
				t.Fatalf("Strategy %q in AllStrategies should be valid", strategy)
			}
		})
	})
}

// =============================================================================
// Property 4: auto-promote 参数验证
// =============================================================================

// TestValidateAutoPromote tests the ValidateAutoPromote function with unit tests
func TestValidateAutoPromote(t *testing.T) {
	tests := []struct {
		name        string
		strategy    cmd.CanaryStrategy
		autoPromote bool
		wantErr     bool
	}{
		{
			name:        "canary75 with auto-promote should succeed",
			strategy:    cmd.Canary75,
			autoPromote: true,
			wantErr:     false,
		},
		{
			name:        "canary75 without auto-promote should succeed",
			strategy:    cmd.Canary75,
			autoPromote: false,
			wantErr:     false,
		},
		{
			name:        "canary10 with auto-promote should fail",
			strategy:    cmd.Canary10,
			autoPromote: true,
			wantErr:     true,
		},
		{
			name:        "canary10 without auto-promote should succeed",
			strategy:    cmd.Canary10,
			autoPromote: false,
			wantErr:     false,
		},
		{
			name:        "canary25 with auto-promote should fail",
			strategy:    cmd.Canary25,
			autoPromote: true,
			wantErr:     true,
		},
		{
			name:        "canary25 without auto-promote should succeed",
			strategy:    cmd.Canary25,
			autoPromote: false,
			wantErr:     false,
		},
		{
			name:        "canary50 with auto-promote should fail",
			strategy:    cmd.Canary50,
			autoPromote: true,
			wantErr:     true,
		},
		{
			name:        "canary50 without auto-promote should succeed",
			strategy:    cmd.Canary50,
			autoPromote: false,
			wantErr:     false,
		},
		{
			name:        "invalid strategy with auto-promote should fail",
			strategy:    cmd.CanaryStrategy("invalid"),
			autoPromote: true,
			wantErr:     true,
		},
		{
			name:        "invalid strategy without auto-promote should succeed",
			strategy:    cmd.CanaryStrategy("invalid"),
			autoPromote: false,
			wantErr:     false,
		},
		{
			name:        "empty strategy with auto-promote should fail",
			strategy:    cmd.CanaryStrategy(""),
			autoPromote: true,
			wantErr:     true,
		},
		{
			name:        "empty strategy without auto-promote should succeed",
			strategy:    cmd.CanaryStrategy(""),
			autoPromote: false,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cmd.ValidateAutoPromote(tt.strategy, tt.autoPromote)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAutoPromote(%q, %v) error = %v, wantErr %v",
					tt.strategy, tt.autoPromote, err, tt.wantErr)
			}
		})
	}
}

// TestProperty4_AutoPromoteValidation tests Property 4: auto-promote 参数验证
// **Validates: Requirements 5.8**
//
// Property 4: For any canary command execution, if --auto-promote is specified
// but the strategy is not canary75, it should return a parameter error.
func TestProperty4_AutoPromoteValidation(t *testing.T) {
	// Property 4a: auto-promote with canary75 should always succeed
	t.Run("auto_promote_with_canary75_succeeds", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			// Always use canary75 strategy with auto-promote=true
			err := cmd.ValidateAutoPromote(cmd.Canary75, true)
			if err != nil {
				t.Fatalf("ValidateAutoPromote(Canary75, true) should succeed, got error: %v", err)
			}
		})
	})

	// Property 4b: auto-promote with non-canary75 strategies should always fail
	t.Run("auto_promote_with_non_canary75_fails", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			// Generate a strategy that is NOT canary75
			nonCanary75Strategies := []cmd.CanaryStrategy{cmd.Canary10, cmd.Canary25, cmd.Canary50}
			strategy := rapid.SampledFrom(nonCanary75Strategies).Draw(t, "strategy")

			err := cmd.ValidateAutoPromote(strategy, true)
			if err == nil {
				t.Fatalf("ValidateAutoPromote(%q, true) should fail for non-canary75 strategy, but got no error", strategy)
			}

			// Verify the error is the expected one
			if err != cmd.ErrAutoPromoteOnlyCanary75 {
				t.Fatalf("ValidateAutoPromote(%q, true) should return ErrAutoPromoteOnlyCanary75, got: %v", strategy, err)
			}
		})
	})

	// Property 4c: without auto-promote, any strategy should succeed
	t.Run("without_auto_promote_any_strategy_succeeds", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			// Generate any valid strategy
			strategyIndex := rapid.IntRange(0, len(cmd.AllStrategies)-1).Draw(t, "strategyIndex")
			strategy := cmd.AllStrategies[strategyIndex]

			err := cmd.ValidateAutoPromote(strategy, false)
			if err != nil {
				t.Fatalf("ValidateAutoPromote(%q, false) should succeed, got error: %v", strategy, err)
			}
		})
	})

	// Property 4d: auto-promote with invalid/random strategies should fail
	t.Run("auto_promote_with_invalid_strategy_fails", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			// Generate a random string that is NOT canary75
			randomStr := rapid.StringMatching(`[a-zA-Z0-9_-]{0,20}`).Draw(t, "randomStr")
			strategy := cmd.CanaryStrategy(randomStr)

			// Skip if we accidentally generated canary75
			if strategy == cmd.Canary75 {
				return
			}

			err := cmd.ValidateAutoPromote(strategy, true)
			if err == nil {
				t.Fatalf("ValidateAutoPromote(%q, true) should fail for non-canary75 strategy, but got no error", strategy)
			}
		})
	})

	// Property 4e: without auto-promote, even invalid strategies should succeed
	t.Run("without_auto_promote_invalid_strategy_succeeds", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			// Generate a random string (could be valid or invalid strategy)
			randomStr := rapid.StringMatching(`[a-zA-Z0-9_-]{0,20}`).Draw(t, "randomStr")
			strategy := cmd.CanaryStrategy(randomStr)

			err := cmd.ValidateAutoPromote(strategy, false)
			if err != nil {
				t.Fatalf("ValidateAutoPromote(%q, false) should succeed regardless of strategy validity, got error: %v", strategy, err)
			}
		})
	})

	// Property 4f: The validation is deterministic - same inputs always produce same outputs
	t.Run("validation_is_deterministic", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			// Generate random strategy and autoPromote value
			strategyIndex := rapid.IntRange(0, len(cmd.AllStrategies)-1).Draw(t, "strategyIndex")
			strategy := cmd.AllStrategies[strategyIndex]
			autoPromote := rapid.Bool().Draw(t, "autoPromote")

			// Call the function twice with the same inputs
			err1 := cmd.ValidateAutoPromote(strategy, autoPromote)
			err2 := cmd.ValidateAutoPromote(strategy, autoPromote)

			// Results should be identical
			if (err1 == nil) != (err2 == nil) {
				t.Fatalf("ValidateAutoPromote(%q, %v) is not deterministic: first call returned %v, second call returned %v",
					strategy, autoPromote, err1, err2)
			}
		})
	})

	// Property 4g: The error message is correct when validation fails
	t.Run("error_message_is_correct", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			// Generate a non-canary75 strategy
			nonCanary75Strategies := []cmd.CanaryStrategy{cmd.Canary10, cmd.Canary25, cmd.Canary50}
			strategy := rapid.SampledFrom(nonCanary75Strategies).Draw(t, "strategy")

			err := cmd.ValidateAutoPromote(strategy, true)
			if err == nil {
				t.Fatalf("Expected error for non-canary75 strategy with auto-promote")
			}

			// Verify error message contains key information
			errMsg := err.Error()
			if errMsg != "--auto-promote 仅在策略为 canary75 时可用" {
				t.Fatalf("Error message should be '--auto-promote 仅在策略为 canary75 时可用', got: %s", errMsg)
			}
		})
	})
}
