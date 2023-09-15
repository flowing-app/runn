package runn

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestBindRunnerRun(t *testing.T) {
	tests := []struct {
		store   store
		cond    map[string]string
		want    store
		wantMap map[string]any
	}{
		{
			store{
				steps: []map[string]any{},
				vars:  map[string]any{},
			},
			map[string]string{},
			store{
				steps: []map[string]any{
					{"run": true},
				},
				vars: map[string]any{},
			},
			map[string]any{
				"steps": []map[string]any{
					{"run": true},
				},
				"vars": map[string]any{},
			},
		},
		{
			store{
				steps: []map[string]any{},
				vars: map[string]any{
					"Key": "value",
				},
				bindVars: map[string]any{},
			},
			map[string]string{
				"newkey": "vars.Key",
			},
			store{
				steps: []map[string]any{
					{"run": true},
				},
				vars: map[string]any{
					"Key": "value",
				},
				bindVars: map[string]any{
					"newkey": "value",
				},
			},
			map[string]any{
				"steps": []map[string]any{
					{"run": true},
				},
				"vars": map[string]any{
					"Key": "value",
				},
				"newkey": "value",
			},
		},
	}
	ctx := context.Background()
	for _, tt := range tests {
		o, err := New()
		if err != nil {
			t.Fatal(err)
		}
		o.store = tt.store
		b, err := newBindRunner(o)
		if err != nil {
			t.Fatal(err)
		}
		if err := b.Run(ctx, tt.cond, true); err != nil {
			t.Fatal(err)
		}

		{
			got := b.operator.store
			opts := []cmp.Option{
				cmp.AllowUnexported(store{}),
			}
			if diff := cmp.Diff(got, tt.want, opts...); diff != "" {
				t.Errorf("%s", diff)
			}
		}

		{
			got := b.operator.store.toMap()
			delete(got, storeEnvKey)
			if diff := cmp.Diff(got, tt.wantMap, nil); diff != "" {
				t.Errorf("%s", diff)
			}
		}
	}
}

func TestBindRunnerRunError(t *testing.T) {
	tests := []struct {
		cond map[string]string
	}{
		{
			map[string]string{
				storeVarsKey: "reverved",
			},
		},
		{
			map[string]string{
				storeStepsKey: "reverved",
			},
		},
		{
			map[string]string{
				storeParentKey: "reverved",
			},
		},
		{
			map[string]string{
				storeIncludedKey: "reverved",
			},
		},
		{
			map[string]string{
				storeCurrentKey: "reverved",
			},
		},
		{
			map[string]string{
				storePreviousKey: "reverved",
			},
		},
		{
			map[string]string{
				loopCountVarKey: "reverved",
			},
		},
	}
	ctx := context.Background()
	for _, tt := range tests {
		o, err := New()
		if err != nil {
			t.Fatal(err)
		}
		b, err := newBindRunner(o)
		if err != nil {
			t.Fatal(err)
		}
		if err := b.Run(ctx, tt.cond, true); err == nil {
			t.Errorf("want error. cond: %v", tt.cond)
		}
	}
}
