package instruction

import "slices"

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
