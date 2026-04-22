package preset

import "github.com/samber/lo"

func appendPresetOption[T any, O ~func(*T)](options []O, opt O) []O {
	if opt == nil {
		return options
	}
	return lo.Concat(options, []O{opt})
}
