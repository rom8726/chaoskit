//go:build !chaos

package chaoskit

import (
	"context"
)

type ChaosContext struct {
}

func NewChaosContext() *ChaosContext {
	return &ChaosContext{}
}

func (*ChaosContext) SetDelayFunc(func() bool) {
}

func (*ChaosContext) SetErrorFunc(func() error) {
}

func (*ChaosContext) SetPanicFunc(func() bool) {
}

func (*ChaosContext) SetNetworkFunc(func(host string, port int) bool) {
}

func (*ChaosContext) SetCancellationFunc(func(context.Context) (context.Context, context.CancelFunc)) {
}

func (*ChaosContext) RegisterProvider(ChaosProvider) {
}

func (*ChaosContext) GetProvider(string) (ChaosProvider, bool) {
	return nil, false
}

func AttachChaos(ctx context.Context, _ *ChaosContext) context.Context {
	return ctx
}

func GetChaos(context.Context) *ChaosContext {
	return nil
}

func MaybeError(context.Context) error {
	return nil
}

func MaybePanic(context.Context) {
}

func MaybeDelay(ctx context.Context) {
}

func MaybeNetworkChaos(context.Context, string, int) {
}

func MaybeCancelContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return ctx, func() {}
}

func ApplyChaos(context.Context, string) bool {
	return false
}
