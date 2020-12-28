package limits

import (
	"encoding/json"

	"github.com/soldatov-s/accp/internal/cache/external"
)

type Limit struct {
	Counter    int
	LastAccess int64 // Unix time
}

type LimitTable map[interface{}]*Limit

func NewLimits(limitCfg map[string]*LimitConfig) map[string]LimitTable {
	l := make(map[string]LimitTable)
	for k := range limitCfg {
		l[k] = make(LimitTable)
	}
	return l
}

func (l *Limit) LoadLimit(name, key string, externalStorage *external.Cache) error {
	if externalStorage != nil {
		if err := externalStorage.JSONGet(key, name+".counter", &l.Counter); err != nil {
			return err
		}
		if err := externalStorage.JSONGet(key, name+".lastaccess", &l.LastAccess); err != nil {
			return err
		}
	}

	return nil
}

func (l *Limit) marshal() (counterData, lastAccessData []byte, err error) {
	if counterData, err = json.Marshal(&l.Counter); err != nil {
		return nil, nil, err
	}

	if lastAccessData, err = json.Marshal(&l.LastAccess); err != nil {
		return nil, nil, err
	}

	return
}

func (l *Limit) UpdateLimit(route, key string, externalStorage *external.Cache) error {
	if externalStorage == nil {
		return nil
	}
	counterData, lastAccessData, err := l.marshal()
	if err != nil {
		return err
	}

	if err := externalStorage.JSONSet(route, key+".counter", string(counterData)); err != nil {
		return err
	}

	if err := externalStorage.JSONSet(route, key+".lastaccess", string(lastAccessData)); err != nil {
		return err
	}

	return nil
}

func (l *Limit) CreateLimit(route, key string, externalStorage *external.Cache) error {
	if externalStorage == nil {
		return nil
	}

	counterData, lastAccessData, err := l.marshal()
	if err != nil {
		return err
	}

	if err := externalStorage.JSONSetNX(route, key+".counter", string(counterData)); err != nil {
		return err
	}

	if err := externalStorage.JSONSetNX(route, key+".lastaccess", string(lastAccessData)); err != nil {
		return err
	}

	return nil
}
