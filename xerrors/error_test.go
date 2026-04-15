package xerrors_test

import (
	"fmt"
	"testing"

	"github.com/xbaseio/xbase/codes"
	"github.com/xbaseio/xbase/xerrors"
)

func TestNew(t *testing.T) {
	innerErr := xerrors.NewCode(
		codes.NewCode(2, "core error"),
		"aaaaaaa",
	)

	err := xerrors.NewCode(
		//"not found",
		codes.NewCode(1, "not found"),
		innerErr.Error(),
	)

	t.Log(err)
	t.Log(err.Code())
	t.Log(err.Next())
	t.Log(err.Cause())
	fmt.Println(fmt.Sprintf("%+v", err))
}
