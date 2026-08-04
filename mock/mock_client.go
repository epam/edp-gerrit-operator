package mock

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"sigs.k8s.io/controller-runtime/pkg/client"

	mock "github.com/epam/edp-common/pkg/mock/controller-runtime/client"
)

type Client struct {
	mock.Client
}

func (c *Client) Update(ctx context.Context, obj client.Object, options ...client.UpdateOption) error {
	called := c.Called()
	parent, ok := called.Get(0).(client.Client)
	if ok {
		return parent.Update(ctx, obj, options...)
	}

	return called.Error(0)
}

func (c *Client) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	called := c.Called(list)
	parent, ok := called.Get(0).(client.Client)
	if ok {
		return parent.List(ctx, list, opts...)
	}

	return called.Error(0)
}

// GroupVersionKindFor and IsObjectNamespaced complete the client.Client
// interface as of controller-runtime v0.15+; tests never call them.
func (c *Client) GroupVersionKindFor(_ runtime.Object) (schema.GroupVersionKind, error) {
	return schema.GroupVersionKind{}, nil
}

func (c *Client) IsObjectNamespaced(_ runtime.Object) (bool, error) {
	return true, nil
}
