package phpgrep

import (
	"github.com/z7zmey/php-parser/freefloating"
	"github.com/z7zmey/php-parser/position"
	"github.com/z7zmey/php-parser/walker"
)

type metaNode struct {
	name string
}

func (metaNode) Walk(v walker.Visitor)                     {}
func (metaNode) Attributes() map[string]interface{}        { return nil }
func (metaNode) SetPosition(p *position.Position)          {}
func (metaNode) GetPosition() *position.Position           { return nil }
func (metaNode) GetFreeFloating() *freefloating.Collection { return nil }

type (
	anyVar struct{ metaNode }
)
