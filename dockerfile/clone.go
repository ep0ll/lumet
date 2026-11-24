package dockerfile

import "slices"

// cloneBase detaches caller-owned option slices from a StageFrom closure.
func cloneBase(base Base) Base {
	switch value := base.(type) {
	case ImageRef:
		value.Options = slices.Clone(value.Options)
		return value
	default:
		return base
	}
}
