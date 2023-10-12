package cache

import (
	"context"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

type mockCache struct {
	*Stats
}

func (m mockCache) Get(ctx context.Context, key string, value any, opts ...Option) error {
	panic("implement me")
}

func (m mockCache) Set(ctx context.Context, key string, value any, opts ...Option) error {
	panic("implement me")
}

func (m mockCache) Has(ctx context.Context, key string) bool {
	panic("implement me")
}

func (m mockCache) Del(ctx context.Context, key string) error {
	panic("implement me")
}

func (m mockCache) IsNotFound(err error) bool {
	return errors.Is(err, ErrCacheMiss)
}

func TestSkipMode(t *testing.T) {
	type args struct {
		mode SkipMode
	}
	tests := []struct {
		name string
		f    SkipMode
		Func func(mode SkipMode) bool
		args args
		want bool
	}{
		{
			name: "in",
			f:    SkipLocal,
			Func: SkipLocal.Is,
			args: args{
				mode: SkipRemote,
			},
			want: false,
		},
		{
			name: "0in",
			f:    SkipMode(0),
			Func: SkipMode(0).Is,
			args: args{
				mode: SkipLocal,
			},
			want: false,
		},
		{
			name: "any",
			f:    SkipLocal,
			Func: SkipLocal.Is,
			args: args{
				mode: SkipMode(0),
			},
			want: false,
		},
		{
			name: "none",
			f:    SkipMode(0),
			Func: func(mode SkipMode) bool {
				return mode.Any()
			},
			args: args{
				mode: SkipMode(0),
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, tt.Func(tt.args.mode), "Is(%v)", tt.args.mode)
		})
	}
}

func TestManager(t *testing.T) {
	type args struct {
		name string
		f    func() Cache
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "default",
			args: args{
				name: "default",
				f: func() Cache {
					return &mockCache{}
				},
			},
		},
		{
			name: "second",
			args: args{
				name: "c1",
				f: func() Cache {
					return &mockCache{}
				},
			},
		},
		{
			name: "duplicate",
			args: args{
				name: "c1",
				f: func() Cache {
					return &mockCache{}
				},
			},
			wantErr: true,
		},
		{
			name: "empty",
			args: args{
				name: "",
				f: func() Cache {
					return &mockCache{}
				},
			},
			wantErr: true,
		},
	}
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.args.f()
			err := RegisterCache(tt.args.name, c)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			_, err = GetCache(tt.args.name + "miss")
			assert.Error(t, err)
			got, _ := GetCache(tt.args.name)
			assert.NotNil(t, got)
			assert.Len(t, _manager.drivers, i+1)
			df, _ := GetCache("default")
			assert.Equal(t, df, _manager.drivers["default"])
		})
	}
}

func TestCacheMethod(t *testing.T) {
	mock := mockCache{}
	t.Run("stats", func(t *testing.T) {
		assert.Nil(t, mock.Stats)
		assert.NotPanics(t, func() {
			mock.Stats.AddHit()
			mock.Stats.AddMiss()
		})
		has := mockCache{&Stats{}}
		has.AddHit()
		has.AddMiss()
		assert.Equal(t, uint64(1), has.Stats.Hits)
		assert.Equal(t, uint64(1), has.Stats.Misses)
	})
	t.Run("cacher", func(t *testing.T) {
		assert.NoError(t, RegisterCache("mock", mock))
		assert.Panics(t, func() {
			_ = Get(context.Background(), "key", nil)
		})
		assert.Panics(t, func() {
			_ = Set(context.Background(), "key", nil)
		})
		assert.Panics(t, func() {
			Has(context.Background(), "key")
		})
		assert.Panics(t, func() {
			_ = Del(context.Background(), "key")
		})
		err := fmt.Errorf("err %w", ErrCacheMiss)
		assert.True(t, IsNotFound(err))
	})
}

func TestOptions(t *testing.T) {
	tests := []struct {
		name    string
		options []Option
		do      func(opts *Options)
	}{
		{
			name:    "ttl min",
			options: []Option{WithTTL(time.Second)},
			do: func(opts *Options) {
				assert.EqualValues(t, time.Second, opts.Expiration())
			},
		},
		{
			name:    "ttl minus",
			options: []Option{WithTTL(time.Second * -1)},
			do: func(opts *Options) {
				assert.EqualValues(t, defaultItemTTL, opts.Expiration())
			},
		},
		{
			name:    "ttl 0",
			options: []Option{WithTTL(0)},
			do: func(opts *Options) {
				assert.EqualValues(t, 0, opts.Expiration())
			},
		},
		{
			name:    "ttl >",
			options: []Option{WithTTL(time.Minute)},
			do: func(opts *Options) {
				assert.EqualValues(t, time.Minute, opts.Expiration())
			},
		},
		{
			name: "getter",
			options: []Option{WithGetter(func(ctx context.Context, key string) (any, error) {
				assert.Equal(t, ctx, context.Background())
				return "value", nil
			})},
			do: func(opts *Options) {
				v, err := opts.Getter(context.Background(), "key")
				assert.NoError(t, err)
				assert.Equal(t, "value", v)
			},
		},
		{
			name:    "With-bool",
			options: []Option{WithSetXX(), WithSetNX(), WithRaw(), WithGroup()},
			do: func(opts *Options) {
				assert.True(t, opts.SetXX)
				assert.True(t, opts.SetNX)
				assert.True(t, opts.Raw)
				assert.True(t, opts.Group)
			},
		},
		{
			name:    "WithSkip",
			options: []Option{WithSkip(SkipRemote)},
			do: func(opts *Options) {
				assert.Equal(t, SkipRemote, opts.Skip)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.do(ApplyOptions(tt.options...))
		})
	}
}

func TestMarshalFunc(t *testing.T) {
	tests := []struct {
		name  string
		data  any
		check func([]byte, any)
	}{
		{
			name: "string",
			data: "string",
			check: func(bt []byte, data any) {
				want := ""
				err := DefaultUnmarshalFunc(bt, &want)
				require.NoError(t, err)
				assert.Equal(t, data, want)
			},
		},
		{
			name: "int",
			data: 1,
			check: func(bt []byte, data any) {
				want := 0
				err := DefaultUnmarshalFunc(bt, &want)
				require.NoError(t, err)
				assert.Equal(t, data, want)
			},
		},
		{
			name: "struct",
			data: struct {
				Name string
			}{
				Name: "name",
			},
			check: func(bt []byte, data any) {
				want := struct {
					Name string
				}{}
				err := DefaultUnmarshalFunc(bt, &want)
				require.NoError(t, err)
				assert.Equal(t, data, want)
			},
		},
		{
			name: "map",
			data: map[string]string{
				"name": "name",
			},
			check: func(bt []byte, data any) {
				want := map[string]string{}
				err := DefaultUnmarshalFunc(bt, &want)
				require.NoError(t, err)
				assert.Equal(t, data, want)
			},
		},
		{
			name: "compress",
			data: func() any {
				var data [64]string
				return data
			}(),
			check: func(bt []byte, data any) {
				want := [64]string{}
				err := DefaultUnmarshalFunc(bt, &want)
				require.NoError(t, err)
				assert.Equal(t, data, want)
			},
		},
		{
			name: "input nil",
			data: nil,
			check: func(bt []byte, data any) {
				err := DefaultUnmarshalFunc(bt, nil)
				assert.NoError(t, err)
			},
		},
		{
			name: "value nil",
			data: []byte("123"),
			check: func(bt []byte, a any) {
				err := DefaultUnmarshalFunc(bt, nil)
				assert.NoError(t, err)
			},
		},
		{
			name: "[]byte",
			data: []byte("[]byte"),
			check: func(bt []byte, data any) {
				var want []byte
				err := DefaultUnmarshalFunc(bt, &want)
				require.NoError(t, err)
				assert.Equal(t, data, want)
				assert.NotSame(t, bt, want)
			},
		},
		{
			name: "empty to has value",
			data: []byte{},
			check: func(bt []byte, data any) {
				want := "123"
				err := DefaultUnmarshalFunc([]byte{}, &want)
				require.NoError(t, err)
				assert.Equal(t, "123", want, "origin value should not be changed")
			},
		},
		{
			name: "unknown compression",
			data: float64(1),
			check: func(bytes []byte, data any) {
				want := float64(1)
				err := DefaultUnmarshalFunc(append(bytes, 1), want)
				assert.Error(t, err, "s2: invalid input")
				err = DefaultUnmarshalFunc(append(bytes, 3), want)
				assert.Error(t, err, "unknown compression method: 3")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bt, err := DefaultMarshalFunc(tt.data)
			assert.NoError(t, err)
			tt.check(bt, tt.data)
		})
	}
}
