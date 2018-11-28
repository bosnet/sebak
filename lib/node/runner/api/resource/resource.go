package resource

import (
	"github.com/nvellon/hal"
)

type Resource interface {
	LinkSelf() string
	Resource() *hal.Resource
	GetMap() hal.Entry
}

type ResourceList struct {
	Resources []Resource
	SelfLink  string
	NextLink  string
	PrevLink  string
}

func NewResourceList(list []Resource, selfLink, nextLink, prevLink string) *ResourceList {
	rl := &ResourceList{
		Resources: list,
		SelfLink:  selfLink,
		NextLink:  nextLink,
		PrevLink:  prevLink,
	}

	return rl
}

func (l ResourceList) Resource() *hal.Resource {
	rl := hal.NewResource(struct{}{}, l.LinkSelf())

	var rCollection hal.ResourceCollection
	for _, apiResource := range l.Resources {
		rCollection = append(rCollection, apiResource.Resource())
	}
	rl.EmbedCollection("records", rCollection)

	if l.LinkPrev() != "" {
		rl.AddLink("prev", hal.NewLink(l.LinkPrev())) //TODO: set prev/next url
	}
	if l.LinkNext() != "" {
		rl.AddLink("next", hal.NewLink(l.LinkNext()))
	}

	return rl
}

func (l ResourceList) LinkSelf() string {
	return l.SelfLink
}

func (l ResourceList) LinkNext() string {
	return l.NextLink
}
func (l ResourceList) LinkPrev() string {
	return l.PrevLink
}

func (l ResourceList) GetMap() hal.Entry {
	return hal.Entry{}
}
