package callctx

import "testing"

func TestToplevelContext(t *testing.T) {
	if ctx1, ctx2 := Toplevel(), Toplevel(); ctx1 != ctx2 {
		t.Errorf("toplevel context is not unique")
	}
}
