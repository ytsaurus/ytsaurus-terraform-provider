package ytsaurus

import (
	"context"
	"fmt"
	"strings"

	"go.ytsaurus.tech/yt/go/ypath"
	"go.ytsaurus.tech/yt/go/yt"
)

type CypressNodeAttributeReader interface {
	GetNode(ctx context.Context, path ypath.YPath, result any, options *yt.GetNodeOptions) error
}

type CypressNodeExistenceChecker interface {
	NodeExists(ctx context.Context, path ypath.YPath, options *yt.NodeExistsOptions) (bool, error)
}

func objectPathByID(id string) ypath.Path {
	return ypath.Path(fmt.Sprintf("#%s", strings.TrimPrefix(id, "#")))
}

func GetObjectByID(ctx context.Context, client yt.Client, id string, result any) error {
	p := objectPathByID(id).Attrs()
	return client.GetNode(ctx, p, result, nil)
}

func ObjectExistsByID(ctx context.Context, client CypressNodeExistenceChecker, id string) (bool, error) {
	if client == nil {
		return false, fmt.Errorf("YTsaurus client is not configured")
	}

	return client.NodeExists(ctx, objectPathByID(id), nil)
}

func CypressNodeIDFromImportID(ctx context.Context, client CypressNodeAttributeReader, importID string, suppressSymlink bool) (string, error) {
	if !strings.HasPrefix(importID, "/") {
		importID = strings.TrimPrefix(importID, "#")
		if suppressSymlink {
			importID = strings.TrimSuffix(importID, "&")
		}
		return importID, nil
	}

	p := ypath.Path(importID)
	if suppressSymlink {
		p = SuppressSymlink(p)
	}

	if client == nil {
		return "", fmt.Errorf("YTsaurus client is not configured")
	}

	var id string
	if err := client.GetNode(ctx, p.Attr("id"), &id, nil); err != nil {
		return "", err
	}

	return id, nil
}

func SuppressSymlink(p ypath.Path) ypath.Path {
	if strings.HasSuffix(p.String(), "&") {
		return p
	}
	return p.SuppressSymlink()
}

func RemoveIfExists(ctx context.Context, client yt.Client, p ypath.Path) error {
	ok, err := client.NodeExists(ctx, p, nil)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	return client.RemoveNode(ctx, p, nil)
}
