package phpgrep

import (
	"github.com/z7zmey/php-parser/freefloating"
	"github.com/z7zmey/php-parser/position"
	"github.com/z7zmey/php-parser/walker"
)

type walkerMethods struct{}

func (walkerMethods) Walk(v walker.Visitor)                     {}
func (walkerMethods) Attributes() map[string]interface{}        { return nil }
func (walkerMethods) SetPosition(p *position.Position)          {}
func (walkerMethods) GetPosition() *position.Position           { return nil }
func (walkerMethods) GetFreeFloating() *freefloating.Collection { return nil }

type metaNode struct {
	walkerMethods
	name string
}

type (
	anyConst struct{ metaNode }
	anyVar   struct{ metaNode }
	anyInt   struct{ metaNode }
	anyFloat struct{ metaNode }
	anyStr   struct{ metaNode }
	anyNum   struct{ metaNode }
	anyExpr  struct{ metaNode }
	anyFunc  struct{ metaNode }
)
