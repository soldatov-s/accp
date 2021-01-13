package accp

import (
	"context"
	"sync"
)

type AccpItem string

const (
	ProvidersItem AccpItem = "providers"
)

type providers struct {
	sync.Map
}

type IProvider interface {
	Start() error
	Shutdown() error
}

func Create(ctx context.Context) (context.Context, *providers) {
	p := &providers{}
	return context.WithValue(ctx, ProvidersItem, p), p
}

func Get(ctx context.Context) *providers {
	v := ctx.Value(ProvidersItem)
	if v != nil {
		return v.(*providers)
	}
	return nil
}

func GetByName(ctx context.Context, name string) interface{} {
	p := Get(ctx)
	if p != nil {
		v, _ := ctx.Value(ProvidersItem).(*providers).Load(name)
		return v
	}

	return nil
}

func RegistrateByName(ctx context.Context, name string, val interface{}) context.Context {
	if v := GetByName(ctx, name); v != nil {
		return ctx
	}

	if v := Get(ctx); v != nil {
		v.Store(name, val)
		return ctx
	}

	c, v := Create(ctx)
	v.Store(name, val)
	return c
}
