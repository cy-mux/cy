package tree

import (
	"strings"
	"unicode"

	"github.com/cfoust/cy/pkg/bind"
	"github.com/cfoust/cy/pkg/params"

	"github.com/sasha-s/go-deadlock"
)

type NodeID = int32

type metaData struct {
	deadlock.RWMutex
	id     NodeID
	name   string
	binds  *bind.BindScope
	params *params.Parameters
}

func (m *metaData) Id() int32 {
	return m.id
}

func (m *metaData) Name() string {
	m.RLock()
	defer m.RUnlock()
	return m.name
}

func (m *metaData) Params() *params.Parameters {
	return m.params
}

func (m *metaData) SetName(name string) {
	m.Lock()
	defer m.Unlock()
	m.name = strings.Map(func(r rune) rune {
		if !unicode.IsPrint(r) || r == '/' {
			return -1
		}

		return r
	}, name)
}

func (m *metaData) Binds() *bind.BindScope {
	return m.binds
}

type Node interface {
	Id() NodeID
	Name() string
	Params() *params.Parameters
	SetName(string)
	Binds() *bind.BindScope
}
