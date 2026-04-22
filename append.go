package clientx

import "github.com/samber/lo"

// AppendHooks appends non-nil hooks to items and preserves order.
func AppendHooks(items []Hook, hooks ...Hook) []Hook {
	return lo.Concat(items, lo.Filter(hooks, func(hook Hook, _ int) bool {
		return hook != nil
	}))
}

// AppendPolicies appends non-nil policies to items and preserves order.
func AppendPolicies(items []Policy, policies ...Policy) []Policy {
	return lo.Concat(items, lo.Filter(policies, func(policy Policy, _ int) bool {
		return policy != nil
	}))
}
