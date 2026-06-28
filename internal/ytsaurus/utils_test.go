package ytsaurus

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"go.ytsaurus.tech/yt/go/ypath"
	"go.ytsaurus.tech/yt/go/yt"
)

type fakeCypressNodeAttributeReader struct {
	id       string
	err      error
	requests []string
}

func (r *fakeCypressNodeAttributeReader) GetNode(_ context.Context, p ypath.YPath, result any, _ *yt.GetNodeOptions) error {
	r.requests = append(r.requests, fmt.Sprint(p))
	if r.err != nil {
		return r.err
	}

	*result.(*string) = r.id
	return nil
}

type fakeCypressNodeExistenceChecker struct {
	ok       bool
	err      error
	requests []string
}

func (c *fakeCypressNodeExistenceChecker) NodeExists(_ context.Context, p ypath.YPath, _ *yt.NodeExistsOptions) (bool, error) {
	c.requests = append(c.requests, fmt.Sprint(p))
	if c.err != nil {
		return false, c.err
	}
	return c.ok, nil
}

func TestCypressNodeIDFromImportID(t *testing.T) {
	ctx := context.Background()

	t.Run("raw ID", func(t *testing.T) {
		got, err := CypressNodeIDFromImportID(ctx, nil, "1-2-3-4", false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "1-2-3-4" {
			t.Fatalf("expected raw ID, got %q", got)
		}
	})

	t.Run("YPath ID", func(t *testing.T) {
		got, err := CypressNodeIDFromImportID(ctx, nil, "#1-2-3-4", false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "1-2-3-4" {
			t.Fatalf("expected ID without #, got %q", got)
		}
	})

	t.Run("path", func(t *testing.T) {
		reader := &fakeCypressNodeAttributeReader{id: "resolved-id"}
		got, err := CypressNodeIDFromImportID(ctx, reader, "//home/project", false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "resolved-id" {
			t.Fatalf("expected resolved ID, got %q", got)
		}
		if len(reader.requests) != 1 || reader.requests[0] != "//home/project/@id" {
			t.Fatalf("unexpected requests: %#v", reader.requests)
		}
	})

	t.Run("link path", func(t *testing.T) {
		reader := &fakeCypressNodeAttributeReader{id: "link-id"}
		got, err := CypressNodeIDFromImportID(ctx, reader, "//home/link", true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "link-id" {
			t.Fatalf("expected resolved link ID, got %q", got)
		}
		if len(reader.requests) != 1 || reader.requests[0] != "//home/link&/@id" {
			t.Fatalf("unexpected requests: %#v", reader.requests)
		}
	})

	t.Run("already suppressed link path", func(t *testing.T) {
		reader := &fakeCypressNodeAttributeReader{id: "link-id"}
		got, err := CypressNodeIDFromImportID(ctx, reader, "//home/link&", true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "link-id" {
			t.Fatalf("expected resolved link ID, got %q", got)
		}
		if len(reader.requests) != 1 || reader.requests[0] != "//home/link&/@id" {
			t.Fatalf("unexpected requests: %#v", reader.requests)
		}
	})

	t.Run("suppressed link ID", func(t *testing.T) {
		got, err := CypressNodeIDFromImportID(ctx, nil, "#1-2-3-4&", true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "1-2-3-4" {
			t.Fatalf("expected link ID without # and &, got %q", got)
		}
	})

	t.Run("path read error", func(t *testing.T) {
		wantErr := errors.New("read failed")
		reader := &fakeCypressNodeAttributeReader{err: wantErr}
		_, err := CypressNodeIDFromImportID(ctx, reader, "//home/project", false)
		if !errors.Is(err, wantErr) {
			t.Fatalf("expected %v, got %v", wantErr, err)
		}
	})
}

func TestObjectExistsByID(t *testing.T) {
	ctx := context.Background()

	t.Run("existing object", func(t *testing.T) {
		checker := &fakeCypressNodeExistenceChecker{ok: true}
		ok, err := ObjectExistsByID(ctx, checker, "1-2-3-4")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !ok {
			t.Fatal("expected object to exist")
		}
		if len(checker.requests) != 1 || checker.requests[0] != "#1-2-3-4" {
			t.Fatalf("unexpected requests: %#v", checker.requests)
		}
	})

	t.Run("missing object", func(t *testing.T) {
		checker := &fakeCypressNodeExistenceChecker{ok: false}
		ok, err := ObjectExistsByID(ctx, checker, "1-2-3-4")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ok {
			t.Fatal("expected object to be missing")
		}
	})

	t.Run("existence check error", func(t *testing.T) {
		wantErr := errors.New("exists failed")
		checker := &fakeCypressNodeExistenceChecker{err: wantErr}
		_, err := ObjectExistsByID(ctx, checker, "1-2-3-4")
		if !errors.Is(err, wantErr) {
			t.Fatalf("expected %v, got %v", wantErr, err)
		}
	})
}
