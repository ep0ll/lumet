package dockerfile

import "context"

func callStageHook(ctx context.Context, hook StageHook, event StageEvent) error {
	if hook == nil {
		return nil
	}
	return hook(ctx, event)
}

