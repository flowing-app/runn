package runn

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/golang-sql/sqlexp/nest"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/k1LoW/httpstub"
	"github.com/k1LoW/runn/testutil"
	"github.com/k1LoW/stopw"
	"github.com/tenntenn/golden"
)

var ErrDummy = errors.New("dummy")

func TestExpand(t *testing.T) {
	tests := []struct {
		steps []map[string]any
		vars  map[string]any
		in    any
		want  any
	}{
		{
			[]map[string]any{},
			map[string]any{},
			map[string]string{"Key": "val"},
			map[string]any{"Key": "val"},
		},
		{
			[]map[string]any{},
			map[string]any{"one": "ichi"},
			map[string]string{"Key": "{{ vars.one }}"},
			map[string]any{"Key": "ichi"},
		},
		{
			[]map[string]any{},
			map[string]any{"one": "ichi"},
			map[string]string{"{{ vars.one }}": "val"},
			map[string]any{"ichi": "val"},
		},
		{
			[]map[string]any{},
			map[string]any{"one": 1},
			map[string]string{"Key": "{{ vars.one }}"},
			map[string]any{"Key": uint64(1)},
		},
		{
			[]map[string]any{},
			map[string]any{"one": 1},
			map[string]string{"Key": "{{ vars.one + 1 }}"},
			map[string]any{"Key": uint64(2)},
		},
		{
			[]map[string]any{},
			map[string]any{"one": 1},
			map[string]string{"Key": "{{ string(vars.one) }}"},
			map[string]any{"Key": "1"},
		},
		{
			[]map[string]any{},
			map[string]any{"one": "01"},
			map[string]string{"path/{{ vars.one }}": "value"},
			map[string]any{"path/01": "value"},
		},
		{
			[]map[string]any{},
			map[string]any{"year": 2022},
			map[string]string{"path?year={{ vars.year }}": "value"},
			map[string]any{"path?year=2022": "value"},
		},
		{
			[]map[string]any{},
			map[string]any{"boolean": true},
			map[string]string{"boolean": "{{ vars.boolean }}"},
			map[string]any{"boolean": true},
		},
		{
			[]map[string]any{},
			map[string]any{"map": map[string]any{"foo": "test", "bar": 1}},
			map[string]string{"map": "{{ vars.map }}"},
			map[string]any{"map": map[string]any{"foo": "test", "bar": uint64(1)}},
		},
		{
			[]map[string]any{},
			map[string]any{"array": []any{map[string]any{"foo": "test1", "bar": 1}, map[string]any{"foo": "test2", "bar": 2}}},
			map[string]string{"array": "{{ vars.array }}"},
			map[string]any{"array": []any{map[string]any{"foo": "test1", "bar": uint64(1)}, map[string]any{"foo": "test2", "bar": uint64(2)}}},
		},
		{
			[]map[string]any{},
			map[string]any{"float": float64(1)},
			map[string]string{"float": "{{ vars.float }}"},
			map[string]any{"float": uint64(1)},
		},
		{
			[]map[string]any{},
			map[string]any{"float": float64(1.01)},
			map[string]string{"float": "{{ vars.float }}"},
			map[string]any{"float": 1.01},
		},
		{
			[]map[string]any{},
			map[string]any{"float": float64(1.00)},
			map[string]string{"float": "{{ vars.float }}"},
			map[string]any{"float": uint64(1)},
		},
		{
			[]map[string]any{},
			map[string]any{"float": float64(-0.9)},
			map[string]string{"float": "{{ vars.float }}"},
			map[string]any{"float": -0.9},
		},
		{
			[]map[string]any{},
			map[string]any{"escape": "C++"},
			map[string]string{"escape": "{{ urlencode(vars.escape) }}"},
			map[string]any{"escape": "C%2B%2B"},
		},
		{
			[]map[string]any{},
			map[string]any{"uint64": uint64(4600)},
			map[string]string{"uint64": "{{ vars.uint64 }}"},
			map[string]any{"uint64": uint64(4600)},
		},
	}
	for _, tt := range tests {
		o, err := New()
		if err != nil {
			t.Fatal(err)
		}
		o.store.steps = tt.steps
		o.store.vars = tt.vars

		got, err := o.expandBeforeRecord(tt.in)
		if err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(got, tt.want, nil); diff != "" {
			t.Errorf("%s", diff)
		}
	}
}

func TestNewOption(t *testing.T) {
	tests := []struct {
		opts    []Option
		wantErr bool
	}{
		{
			[]Option{Book("testdata/book/book.yml"), Runner("db", "sqlite://path/to/test.db")},
			false,
		},
		{
			[]Option{Runner("db", "sqlite://path/to/test.db"), Book("testdata/book/book.yml")},
			false,
		},
		{
			[]Option{Book("testdata/book/notfound.yml")},
			true,
		},
		{
			[]Option{Runner("db", "unsupported://hostname")},
			true,
		},
		{
			[]Option{Runner("db", "sqlite://path/to/test.db"), HTTPRunner("db", "https://api.github.com", nil)},
			true,
		},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%v", i), func(t *testing.T) {
			_, err := New(tt.opts...)
			got := (err != nil)
			if got != tt.wantErr {
				t.Errorf("got %v\nwant %v", got, tt.wantErr)
			}
		})
	}
}

func TestRun(t *testing.T) {
	tests := []struct {
		book string
	}{
		{"testdata/book/db.yml"},
		{"testdata/book/only_if_included.yml"},
		{"testdata/book/if.yml"},
		{"testdata/book/previous.yml"},
		{"testdata/book/faker.yml"},
		{"testdata/book/env.yml"},
		{"testdata/book/builtin-json.yml"},
	}
	ctx := context.Background()
	t.Setenv("DEBUG", "false")
	for _, tt := range tests {
		tt := tt
		t.Run(tt.book, func(t *testing.T) {
			t.Parallel()
			db, _ := testutil.SQLite(t)
			o, err := New(Book(tt.book), DBRunner("db", db))
			if err != nil {
				t.Fatal(err)
			}
			if err := o.Run(ctx); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestRunAsT(t *testing.T) {
	tests := []struct {
		book string
	}{
		{"testdata/book/db.yml"},
		{"testdata/book/only_if_included.yml"},
		{"testdata/book/if.yml"},
		{"testdata/book/previous.yml"},
	}
	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.book, func(t *testing.T) {
			db, _ := testutil.SQLite(t)
			o, err := New(Book(tt.book), DBRunner("db", db))
			if err != nil {
				t.Fatal(err)
			}
			if err := o.Run(ctx); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestRunUsingLoop(t *testing.T) {
	ts := httpstub.NewServer(t)
	counter := 0
	ts.Method(http.MethodGet).Handler(func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte(fmt.Sprintf("%d", counter))); err != nil {
			t.Fatal(err)
		}
		counter += 1
	})
	t.Cleanup(func() {
		ts.Close()
	})

	tests := []struct {
		book string
	}{
		{"testdata/book/loop.yml"},
	}
	ctx := context.Background()
	for _, tt := range tests {
		o, err := New(T(t), Book(tt.book), Runner("req", ts.Server().URL))
		if err != nil {
			t.Fatal(err)
		}
		if err := o.Run(ctx); err != nil {
			t.Error(err)
		}
	}
}

func TestLoad(t *testing.T) {
	tests := []struct {
		paths    string
		RUNN_RUN string
		RUNN_ID  string
		want     int
	}{
		{
			"testdata/book/**/*",
			"",
			"",
			func() int {
				e, err := os.ReadDir("testdata/book/")
				if err != nil {
					t.Fatal(err)
				}
				return len(e)
			}(),
		},
		{"testdata/book/**/*", "initdb", "", 1},
		{"testdata/book/**/*", "nonexistent", "", 0},
		{"testdata/book/**/*", "", "eb33c9aed04a7f1e03c1a1246b5d7bdaefd903d3", 1},
		{"testdata/book/**/*", "", "eb33c9a", 1},
	}
	for _, tt := range tests {
		t.Setenv("RUNN_RUN", tt.RUNN_RUN)
		t.Setenv("RUNN_ID", tt.RUNN_ID)
		opts := []Option{
			Runner("req", "https://api.github.com"),
			Runner("db", "sqlite://path/to/test.db"),
			SSHRunner("sc", testutil.NewNullSSHClient()),
			SSHRunner("sc2", testutil.NewNullSSHClient()),
			SSHRunner("sc3", testutil.NewNullSSHClient()),
		}
		ops, err := Load(tt.paths, opts...)
		if err != nil {
			t.Fatal(err)
		}
		got := len(ops.ops)
		if got != tt.want {
			t.Errorf("got %v\nwant %v", got, tt.want)
		}
	}
}

func TestLoadOnly(t *testing.T) {
	t.Run("Allow to load somewhat broken runbooks", func(t *testing.T) {
		_, err := Load("testdata/book/**/*", LoadOnly())
		if err != nil {
			t.Error(err)
		}
	})
}

func TestRunN(t *testing.T) {
	tests := []struct {
		paths    string
		RUNN_RUN string
		failFast bool
		want     *runNResult
	}{
		{"testdata/book/runn_*", "", false, newRunNResult(t, 4, []*RunResult{
			{
				ID:          "84ff32ce475541124d3b28efcecb11268d79f2c6",
				Path:        "testdata/book/runn_0_success.yml",
				Err:         nil,
				StepResults: []*StepResult{{Key: "0", Err: nil}},
			},
			{
				ID:          "b6d90c331b04ab198ca95b13c5f656fd2522e53b",
				Path:        "testdata/book/runn_1_fail.yml",
				Err:         ErrDummy,
				StepResults: []*StepResult{{Key: "0", Err: ErrDummy}},
			},
			{
				ID:          "faeec884c284f9c2527f840372fc01ed8351a377",
				Path:        "testdata/book/runn_2_success.yml",
				Err:         nil,
				StepResults: []*StepResult{{Key: "0", Err: nil}},
			},
			{
				ID:          "15519f515b984b9b25dae1cfde43597cd035dc3d",
				Path:        "testdata/book/runn_3.skip.yml",
				Err:         nil,
				Skipped:     true,
				StepResults: []*StepResult{{Key: "0", Err: nil, Skipped: true}},
			},
		})},
		{"testdata/book/runn_*", "", true, newRunNResult(t, 4, []*RunResult{
			{
				ID:          "84ff32ce475541124d3b28efcecb11268d79f2c6",
				Path:        "testdata/book/runn_0_success.yml",
				Err:         nil,
				StepResults: []*StepResult{{Key: "0", Err: nil}},
			},
			{
				ID:          "b6d90c331b04ab198ca95b13c5f656fd2522e53b",
				Path:        "testdata/book/runn_1_fail.yml",
				Err:         ErrDummy,
				StepResults: []*StepResult{{Key: "0", Err: ErrDummy}},
			},
		})},
		{"testdata/book/runn_*", "runn_0", false, newRunNResult(t, 1, []*RunResult{
			{
				ID:          "84ff32ce475541124d3b28efcecb11268d79f2c6",
				Path:        "testdata/book/runn_0_success.yml",
				Err:         nil,
				StepResults: []*StepResult{{Key: "0", Err: nil}},
			},
		})},
	}
	ctx := context.Background()
	for _, tt := range tests {
		t.Setenv("RUNN_RUN", tt.RUNN_RUN)
		ops, err := Load(tt.paths, FailFast(tt.failFast))
		if err != nil {
			t.Fatal(err)
		}
		_ = ops.RunN(ctx)
		got := ops.Result().Simplify()
		want := tt.want.Simplify()
		if diff := cmp.Diff(got, want, nil); diff != "" {
			t.Errorf("%s", diff)
		}
	}
}

func TestSkipIncluded(t *testing.T) {
	tests := []struct {
		paths        string
		skipIncluded bool
		want         int
	}{
		{"testdata/book/include_*", false, 3},
		{"testdata/book/include_*", true, 1},
	}
	for _, tt := range tests {
		ops, err := Load(tt.paths, SkipIncluded(tt.skipIncluded), Runner("req", "https://api.github.com"), Runner("db", "sqlite://path/to/test.db"))
		if err != nil {
			t.Fatal(err)
		}
		got := len(ops.ops)
		if got != tt.want {
			t.Errorf("got %v\nwant %v", got, tt.want)
		}
	}
}

func TestSkipTest(t *testing.T) {
	tests := []struct {
		book string
	}{
		{"testdata/book/skip_test.yml"},
	}
	ctx := context.Background()
	for _, tt := range tests {
		o, err := New(Book(tt.book))
		if err != nil {
			t.Fatal(err)
		}
		if err := o.Run(ctx); err != nil {
			t.Error(err)
		}
	}
}

func TestHookFuncTest(t *testing.T) {
	count := 0
	tests := []struct {
		book        string
		beforeFuncs []func(*RunResult) error
		afterFuncs  []func(*RunResult) error
		want        int
	}{
		{"testdata/book/skip_test.yml", nil, nil, 0},
		{
			"testdata/book/skip_test.yml",
			[]func(*RunResult) error{
				func(*RunResult) error {
					count += 3
					return nil
				},
				func(*RunResult) error {
					count = count * 2
					return nil
				},
			},
			[]func(*RunResult) error{
				func(*RunResult) error {
					count += 7
					return nil
				},
			},
			13,
		},
	}
	ctx := context.Background()
	for _, tt := range tests {
		count = 0
		opts := []Option{
			Book(tt.book),
		}
		for _, fn := range tt.beforeFuncs {
			opts = append(opts, BeforeFunc(fn))
		}
		for _, fn := range tt.afterFuncs {
			opts = append(opts, AfterFunc(fn))
		}
		o, err := New(opts...)
		if err != nil {
			t.Fatal(err)
		}
		if err := o.Run(ctx); err != nil {
			t.Error(err)
		}
		if count != tt.want {
			t.Errorf("got %v\nwant %v", count, tt.want)
		}
	}
}

func TestInclude(t *testing.T) {
	tests := []struct {
		book string
	}{
		{"testdata/book/include_main.yml"},
	}
	ctx := context.Background()
	for _, tt := range tests {
		o, err := New(Book(tt.book), Func("upcase", strings.ToUpper))
		if err != nil {
			t.Fatal(err)
		}
		if err := o.Run(ctx); err != nil {
			t.Error(err)
		}
	}
}

func TestDump(t *testing.T) {
	tests := []struct {
		book string
	}{
		{"testdata/book/dump.yml"},
	}
	ctx := context.Background()
	for _, tt := range tests {
		o, err := New(Book(tt.book), Func("upcase", strings.ToUpper), Stdout(io.Discard), Stderr(io.Discard))
		if err != nil {
			t.Fatal(err)
		}
		if err := o.Run(ctx); err != nil {
			t.Error(err)
		}
	}
}

func TestShard(t *testing.T) {
	tests := []struct {
		n int
	}{
		{2}, {3}, {4}, {5}, {6}, {7}, {11}, {13}, {17}, {99},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("n=%d", tt.n), func(t *testing.T) {
			got := []*operator{}
			opts := []Option{
				Runner("req", "https://api.github.com"),
				Runner("db", "sqlite://path/to/test.db"),
				SSHRunner("sc", testutil.NewNullSSHClient()),
				SSHRunner("sc2", testutil.NewNullSSHClient()),
				SSHRunner("sc3", testutil.NewNullSSHClient()),
			}
			all, err := Load("testdata/book/**/*", opts...)
			if err != nil {
				t.Fatal(err)
			}
			sortOperators(all.ops)
			want := all.ops
			for i := 0; i < tt.n; i++ {
				ops, err := Load("testdata/book/**/*", append(opts, RunShard(tt.n, i))...)
				if err != nil {
					t.Fatal(err)
				}
				selected, err := ops.SelectedOperators()
				if err != nil {
					t.Fatal(err)
				}
				got = append(got, selected...)
			}
			if len(got) != len(want) {
				t.Errorf("got %v\nwant %v", len(got), len(want))
			}
			sortOperators(got)
			allow := []any{
				operator{}, httpRunner{}, dbRunner{}, grpcRunner{}, cdpRunner{}, sshRunner{},
			}
			ignore := []any{
				Step{}, store{}, sql.DB{}, os.File{}, stopw.Span{}, debugger{}, nest.DB{}, Loop{},
			}
			dopts := []cmp.Option{
				cmp.AllowUnexported(allow...),
				cmpopts.IgnoreUnexported(ignore...),
				cmpopts.IgnoreFields(stopw.Span{}, "ID"),
				cmpopts.IgnoreFields(operator{}, "id"),
				cmpopts.IgnoreFields(operator{}, "concurrency"),
				cmpopts.IgnoreFields(operator{}, "mu"),
				cmpopts.IgnoreFields(cdpRunner{}, "ctx"),
				cmpopts.IgnoreFields(cdpRunner{}, "cancel"),
				cmpopts.IgnoreFields(cdpRunner{}, "opts"),
				cmpopts.IgnoreFields(sshRunner{}, "client"),
				cmpopts.IgnoreFields(sshRunner{}, "sess"),
				cmpopts.IgnoreFields(sshRunner{}, "stdin"),
				cmpopts.IgnoreFields(sshRunner{}, "stdout"),
				cmpopts.IgnoreFields(sshRunner{}, "stderr"),
				cmpopts.IgnoreFields(http.Client{}, "Transport"),
			}
			if diff := cmp.Diff(got, want, dopts...); diff != "" {
				t.Errorf("%s", diff)
			}
		})
	}
}

func TestVars(t *testing.T) {
	tests := []struct {
		opts    []Option
		wantErr bool
	}{
		{
			[]Option{Book("testdata/book/vars.yml"), Var("token", "world")},
			false,
		},
		{
			[]Option{Book("testdata/book/vars.yml")},
			true,
		},
		{
			[]Option{Book("testdata/book/vars_external.yml"), Var("override", "json://../vars.json")},
			false,
		},
		{
			[]Option{Book("testdata/book/vars_external.yml")},
			true,
		},
	}
	ctx := context.Background()
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			o, err := New(tt.opts...)
			if err != nil {
				t.Error(err)
			}
			if err := o.Run(ctx); err != nil {
				if !tt.wantErr {
					t.Errorf("got %v\n", err)
				}
				return
			}
			if tt.wantErr {
				t.Error("want error")
			}
		})
	}
}

func TestHttp(t *testing.T) {
	tests := []struct {
		book string
	}{
		{"testdata/book/http.yml"},
		{"testdata/book/http_not_follow_redirect.yml"},
		{"testdata/book/http_with_json.yml"},
	}
	ctx := context.Background()
	for _, tt := range tests {
		tt := tt
		t.Run(tt.book, func(t *testing.T) {
			ts := testutil.HTTPServer(t)
			t.Setenv("TEST_HTTP_END_POINT", ts.URL)
			o, err := New(Book(tt.book))
			if err != nil {
				t.Fatal(err)
			}
			if err := o.Run(ctx); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestGrpc(t *testing.T) {
	tests := []struct {
		book string
	}{
		{"testdata/book/grpc.yml"},
		{"testdata/book/grpc_with_json.yml"},
	}
	ctx := context.Background()
	for _, tt := range tests {
		tt := tt
		t.Run(tt.book, func(t *testing.T) {
			t.Parallel()
			ts := testutil.GRPCServer(t, false, false)
			o, err := New(Book(tt.book), GrpcRunner("greq", ts.Conn()))
			if err != nil {
				t.Fatal(err)
			}
			if err := o.Run(ctx); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestAfterFuncAlwaysCall(t *testing.T) {
	tests := []struct {
		book    string
		wantErr bool
	}{
		{"testdata/book/always_success.yml", false},
		{"testdata/book/always_failure.yml", true},
	}
	ctx := context.Background()
	for _, tt := range tests {
		tt := tt
		t.Run(tt.book, func(t *testing.T) {
			var rerr error
			called := false
			o, err := New(Book(tt.book), AfterFunc(func(rr *RunResult) error {
				if rr != nil {
					rerr = rr.Err
				}
				called = true
				return nil
			}))
			if err != nil {
				t.Fatal(err)
			}
			rrerr := o.Run(ctx)
			if (rrerr == nil) != (rerr == nil) {
				t.Errorf("o.Run(ctx) error: %v\nafterFunc got error:%v", rrerr, rerr)
			}
			if !called {
				t.Error("called should be true")
			}
			if rerr != nil && !tt.wantErr {
				t.Errorf("got err: %s", err)
			}
			if rerr == nil && tt.wantErr {
				t.Error("want err")
			}
		})
	}
}

func TestBeforeFuncErr(t *testing.T) {
	tests := []struct {
		book string
	}{
		{"testdata/book/always_success.yml"},
		{"testdata/book/always_failure.yml"},
	}
	ctx := context.Background()
	for _, tt := range tests {
		tt := tt
		t.Run(tt.book, func(t *testing.T) {
			o, err := New(Book(tt.book), BeforeFunc(func(*RunResult) error {
				return errors.New("before func error")
			}))
			if err != nil {
				t.Fatal(err)
			}
			if got := o.Run(ctx); got != nil {
				if errors.As(got, &BeforeFuncError{}) {
					t.Errorf("got %v\nwant %T", got, &BeforeFuncError{})
				}
				return
			}
			t.Error("want err")
		})
	}
}

func TestAfterFuncErr(t *testing.T) {
	tests := []struct {
		book string
	}{
		{"testdata/book/always_success.yml"},
		{"testdata/book/always_failure.yml"},
	}
	ctx := context.Background()
	for _, tt := range tests {
		tt := tt
		t.Run(tt.book, func(t *testing.T) {
			o, err := New(Book(tt.book), AfterFunc(func(*RunResult) error {
				return errors.New("after func error")
			}))
			if err != nil {
				t.Fatal(err)
			}
			if got := o.Run(ctx); got != nil {
				if errors.As(got, &AfterFuncError{}) {
					t.Errorf("got %v\nwant %T", got, &AfterFuncError{})
				}
				return
			}
			t.Error("want err")
		})
	}
}

func TestAfterFuncIf(t *testing.T) {
	tests := []struct {
		book    string
		ifCond  string
		wantErr bool
	}{
		{"testdata/book/always_success.yml", "true", true},
		{"testdata/book/always_failure.yml", "true", true},
		{"testdata/book/always_success.yml", "false", false},
		{"testdata/book/always_failure.yml", "false", true},
	}
	ctx := context.Background()
	for _, tt := range tests {
		tt := tt
		t.Run(tt.book, func(t *testing.T) {
			o, err := New(Book(tt.book), AfterFuncIf(func(*RunResult) error {
				return errors.New("after func error")
			}, tt.ifCond))
			if err != nil {
				t.Fatal(err)
			}
			if err := o.Run(ctx); err != nil {
				if !tt.wantErr {
					t.Errorf("got %v\nwant nil", err)
				}
				return
			}
			if tt.wantErr {
				t.Error("want err")
			}
		})
	}
}

func TestStoreKeys(t *testing.T) {
	tests := []struct {
		book string
	}{
		{"testdata/book/store_keys.yml"},
	}
	ctx := context.Background()
	for _, tt := range tests {
		tt := tt
		t.Run(tt.book, func(t *testing.T) {
			ts := testutil.HTTPServer(t)
			t.Setenv("TEST_HTTP_END_POINT", ts.URL)
			o, err := New(Book(tt.book))
			if err != nil {
				t.Fatal(err)
			}
			if err := o.Run(ctx); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestLoop(t *testing.T) {
	tests := []struct {
		book    string
		count   int
		wantErr bool
	}{
		{"testdata/book/rootloop.yml", 10, false},
		{"testdata/book/rootloop.yml", 5, true},
		{"testdata/book/rootlooponly.yml", 5, false},
		{"testdata/book/rootlooponly.yml", 6, true},
	}
	ctx := context.Background()
	for i, tt := range tests {
		tt := tt
		t.Run(tt.book, func(t *testing.T) {
			key := fmt.Sprintf("testloop_count%d", i)
			got := new(bytes.Buffer)
			o, err := New(Book(tt.book), Var("lcount", tt.count), Stdout(got))
			if err != nil {
				t.Fatal(err)
			}
			if err := o.Run(ctx); err != nil {
				if !tt.wantErr {
					t.Errorf("got err: %v", err)
				}
			} else {
				if tt.wantErr {
					t.Error("want err")
				}
			}
			if os.Getenv("UPDATE_GOLDEN") != "" {
				golden.Update(t, "testdata", key, got)
				return
			}
			if diff := golden.Diff(t, "testdata", key, got); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func TestFailWithStepDesc(t *testing.T) {
	tests := []struct {
		book              string
		expectedSubString string
	}{
		{
			book:              "testdata/book/failure_with_step_desc.yml",
			expectedSubString: "this is description",
		},
	}
	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.book, func(t *testing.T) {
			out := new(bytes.Buffer)
			opts := []Option{
				Book(tt.book),
				Stderr(out),
			}
			o, err := New(opts...)
			if err != nil {
				t.Fatal(err)
			}
			err = o.Run(ctx)

			if !strings.Contains(err.Error(), tt.expectedSubString) {
				t.Errorf("expected: \"%s\" is contained in result but not.\ngot string: %s", tt.expectedSubString, err.Error())
			}
		})
	}
}

func TestStepResult(t *testing.T) {
	tests := []struct {
		book  string
		force bool
		want  []*StepResult
	}{
		{"testdata/book/always_success.yml", false, []*StepResult{{Skipped: false, Err: nil}, {Skipped: false, Err: nil}, {Skipped: false, Err: nil}}},
		{"testdata/book/always_failure.yml", false, []*StepResult{{Skipped: false, Err: nil}, {Skipped: false, Err: errors.New("some error")}, {Skipped: true, Err: nil}}},
		{"testdata/book/skip_test.yml", false, []*StepResult{{Skipped: true, Err: nil}, {Skipped: false, Err: nil}}},
		{"testdata/book/only_if_included.yml", false, []*StepResult{{Skipped: true, Err: nil}, {Skipped: true, Err: nil}}},
		{"testdata/book/force.yml", false, []*StepResult{{Skipped: false, Err: nil}, {Skipped: false, Err: errors.New("some error")}, {Skipped: false, Err: nil}}},
		{"testdata/book/always_failure.yml", true, []*StepResult{{Skipped: false, Err: nil}, {Skipped: false, Err: errors.New("some error")}, {Skipped: false, Err: nil}}},
		{"testdata/book/only_if_included.yml", true, []*StepResult{{Skipped: true, Err: nil}, {Skipped: true, Err: nil}}},
	}
	ctx := context.Background()
	for _, tt := range tests {
		tt := tt
		t.Run(tt.book, func(t *testing.T) {
			o, err := New(Book(tt.book), Force(tt.force))
			if err != nil {
				t.Fatal(err)
			}
			_ = o.Run(ctx)
			for i, s := range o.steps {
				got := s.result
				if got == nil {
					t.Errorf("want Step[%d] result", i)
					continue
				}
				want := tt.want[i]
				if got.Skipped != want.Skipped {
					t.Errorf("Step[%d] got %v\nwant %v", i, got.Skipped, want.Skipped)
					continue
				}
				if (got.Err == nil) != (want.Err == nil) {
					t.Errorf("Step[%d] got %v\nwant %v", i, got.Err, want.Err)
					continue
				}
			}
		})
	}
}

func TestStepOutcome(t *testing.T) {
	tests := []struct {
		book  string
		force bool
		want  []result
	}{
		{"testdata/book/always_success.yml", false, []result{resultSuccess, resultSuccess, resultSuccess}},
		{"testdata/book/always_failure.yml", false, []result{resultSuccess, resultFailure, resultSkipped}},
		{"testdata/book/skip_test.yml", false, []result{resultSkipped, resultSuccess}},
		{"testdata/book/only_if_included.yml", false, []result{resultSkipped, resultSkipped}},
		{"testdata/book/always_failure.yml", true, []result{resultSuccess, resultFailure, resultSuccess}},
		{"testdata/book/only_if_included.yml", true, []result{resultSkipped, resultSkipped}},
	}
	ctx := context.Background()
	for _, tt := range tests {
		tt := tt
		t.Run(tt.book, func(t *testing.T) {
			o, err := New(Book(tt.book), Force(tt.force))
			if err != nil {
				t.Fatal(err)
			}
			_ = o.Run(ctx)
			if o.useMap {
				if len(o.store.stepMapKeys) != len(tt.want) {
					t.Errorf("got %v\nwant %v", len(o.store.stepMapKeys), len(tt.want))
				}
				i := 0
				for _, k := range o.store.stepMapKeys {
					got, ok := o.store.stepMap[k][storeOutcomeKey]
					if !ok {
						t.Error("want outcome")
						continue
					}
					want := tt.want[i]
					if got != want {
						t.Errorf("Step[%d] got %v\nwant %v", i, got, want)
					}
					i++
				}
			} else {
				if len(o.store.steps) != len(tt.want) {
					t.Errorf("got %v\nwant %v", len(o.store.steps), len(tt.want))
				}
				for i, s := range o.store.steps {
					got, ok := s[storeOutcomeKey]
					if !ok {
						t.Error("want outcome")
						continue
					}
					want := tt.want[i]
					if got != want {
						t.Errorf("Step[%d] got %v\nwant %v", i, got, want)
					}
				}
			}
		})
	}
}

func TestRunnerRenew(t *testing.T) {
	book := "testdata/book/cdploop.yml"
	ctx := context.Background()
	ts := testutil.HTTPServer(t)
	var (
		o    *operator
		err  error
		once sync.Once
	)
	opts := []Option{
		Book(book),
		Var("url", ts.URL),
		BeforeFunc(func(*RunResult) error {
			once.Do(func() {
				// Close the runner connections for the first time only to get an error
				for _, r := range o.cdpRunners {
					if err := r.Close(); err != nil {
						t.Fatal(err)
					}
				}
			})
			return nil
		}),
	}
	o, err = New(opts...)
	if err != nil {
		t.Fatal(err)
	}
	if err := o.Run(ctx); err != nil {
		t.Error(err)
	}
}

func newRunNResult(t *testing.T, total int64, results []*RunResult) *runNResult {
	r := &runNResult{}
	r.Total.Store(total)
	r.RunResults = results
	return r
}
