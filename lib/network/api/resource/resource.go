package resource

import (
	"github.com/nvellon/hal"
)

type APIResource interface {
	LinkSelf() string
	Resource() *hal.Resource
	GetMap() hal.Entry
}

type ResourceList struct {
	Resources []APIResource
	SelfLink  string
}

func (l ResourceList) Resource() *hal.Resource {
	rl := hal.NewResource(struct{}{}, l.LinkSelf())
	for _, apiResource := range l.Resources {
		r := apiResource.Resource()
		rl.Embed("records", r)
	}
	rl.AddLink("prev", hal.NewLink(l.LinkSelf())) //TODO: set prev/next url
	rl.AddLink("next", hal.NewLink(l.LinkSelf()))

	return rl
}

func (l ResourceList) LinkSelf() string {
	return l.SelfLink
}
func (l ResourceList) GetMap() hal.Entry {
	return hal.Entry{}
}
