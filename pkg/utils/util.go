package utils

import (
	"context"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/generic/registry"
	"strconv"
)

func ParseInt642Str(crv int64) string {
	return strconv.FormatInt(crv, 10)
}

func ParseStr2Int64(crvStr string) (int64, error) {
	crv, err := strconv.ParseInt(crvStr, 10, 64)
	if err != nil {
		return 0, err
	}
	return crv, nil
}

func IsEqual(crvStr1 string, crvStr2 string) bool {
	return crvStr1 == crvStr2
}

type keyFunc func(runtime.Object) (string, error)

func GetKeyFunc(gvr schema.GroupVersionResource, isNamespaced bool) keyFunc {
	prefix := gvr.Group + "/" + gvr.Resource

	var KeyFunc func(ctx context.Context, name string) (string, error)
	if isNamespaced {
		KeyFunc = func(ctx context.Context, name string) (string, error) {
			return registry.NamespaceKeyFunc(ctx, prefix, name)
		}
	} else {
		KeyFunc = func(ctx context.Context, name string) (string, error) {
			return registry.NoNamespaceKeyFunc(ctx, prefix, name)
		}
	}

	// We adapt the store's keyFunc so that we can use it with the StorageDecorator
	// without making any assumptions about where objects are stored in etcd
	kc := func(obj runtime.Object) (string, error) {
		accessor, err := meta.Accessor(obj)
		if err != nil {
			return "", err
		}

		if isNamespaced {
			return KeyFunc(genericapirequest.WithNamespace(genericapirequest.NewContext(), accessor.GetNamespace()), accessor.GetName())
		}

		return KeyFunc(genericapirequest.NewContext(), accessor.GetName())
	}

	return kc
}
