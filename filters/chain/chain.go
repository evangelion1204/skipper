package chain

import (
	"github.com/zalando/skipper/filters"
)

type filter struct {
	filters []filters.Spec
	name string
}

type spec struct {
	filters []filters.Spec
	name string
}

func NewSpec(name string) (filters.Spec, error) {
	return spec{
		name: name,
		filters: filters.Spec{},
	}, nil
}

func (chain filter) Request(ctx filters.FilterContext) {
	for i, filter := range chain.filters {
		filter.Request(ctx)
		if ctx.Served() {
			ctx.StateBag()["fashionStoreExitIndex"] = i
			return
		}
	}
}

func (chain filter) Response(ctx filters.FilterContext) {
	if index, ok := ctx.StateBag()["fashionStoreExitIndex"]; ok {
		chain.filters = chain.filters[:index.(int)]
	}
	for _, filter := range chain.filters {
		filter.Response(ctx)
	}
}

func (s spec) Name() string {
	return spec.name
}

func (s spec) CreateFilter(config []interface{}) (filters.Filter, error) {
	filters := filter{
		name: s.name,
		filters: make([]filters.Spec, len(s.filters)),
	}

	for i, sp := range s.filters {
		filter, err := sp.CreateFilter([]interface{}{})

		if err != nil {
			return nil, err
		}

		filters[i] = filter
	}

	return filters, nil
}
