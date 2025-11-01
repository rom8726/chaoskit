package injectors

import (
	"context"
	"errors"
	"testing"
)

type fakeInj struct {
	name      string
	injectErr error
	stopErr   error
	injected  *int
}

func (f *fakeInj) Name() string { return f.name }
func (f *fakeInj) Inject(ctx context.Context) error {
	if f.injected != nil {
		*f.injected++
	}
	return f.injectErr
}
func (f *fakeInj) Stop(ctx context.Context) error { return f.stopErr }

func TestComposite_Inject_OrderAndError(t *testing.T) {
	cnt := 0
	a := &fakeInj{name: "A", injected: &cnt}
	b := &fakeInj{name: "B", injected: &cnt, injectErr: errors.New("boom")}

	c := Composite("combo", a, b)
	err := c.Inject(context.Background())
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if cnt != 2 {
		t.Fatalf("expected both injectors called, got count=%d", cnt)
	}
}

func TestComposite_Stop_AggregatesErrors(t *testing.T) {
	a := &fakeInj{name: "A", stopErr: errors.New("e1")}
	b := &fakeInj{name: "B", stopErr: errors.New("e2")}

	c := Composite("combo", a, b)
	err := c.Stop(context.Background())
	if err == nil {
		t.Fatalf("expected aggregated error, got nil")
	}
}
