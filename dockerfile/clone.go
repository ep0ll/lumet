package dockerfile

import "slices"

func cloneBase(base Base) Base {
	switch value := base.(type) {
	case ImageRef:
		value.Options = slices.Clone(value.Options)
		return value
	default:
		return base
	}
}

func cloneCopySource(source CopySource) CopySource {
	switch value := source.(type) {
	case ImageRef:
		value.Options = slices.Clone(value.Options)
		return value
	default:
		return source
	}
}

func cloneMount(mount Mount) Mount {
	switch value := mount.(type) {
	case BindMount:
		value.From = cloneCopySource(value.From)
		return value
	case CacheMount:
		value.From = cloneCopySource(value.From)
		return value
	default:
		return mount
	}
}

func cloneCommand(command Command) Command {
	switch value := command.(type) {
	case ExecForm:
		return slices.Clone(value)
	default:
		return command
	}
}
