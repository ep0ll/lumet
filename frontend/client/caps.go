package client

import (
	"github.com/moby/buildkit/solver/pb"
	"github.com/moby/buildkit/util/apicaps"
	"github.com/pkg/errors"
)

func validateMinCaps(llbCaps apicaps.CapSet) error {
	if err := llbCaps.Supports(pb.CapFileBase); err != nil {
		return errors.Wrap(err, "needs BuildKit 0.5 or later")
	}
	return nil
}
