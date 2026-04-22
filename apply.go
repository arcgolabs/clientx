package clientx

import "github.com/DaiYuANg/arcgo/pkg/option"

// Apply applies non-nil function options to target.
func Apply[T any, O ~func(*T)](target *T, opts ...O) {
	option.Apply(target, opts...)
}
