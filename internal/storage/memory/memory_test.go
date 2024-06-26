package memorystorage

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

var MemoryStorage Storage
var TestItem1 = StorageItem{
	Link:        "url",
	ShortLink:   "shortUrl",
	ID:          "1",
	DeletedFlag: false,
}

var TestItem2 = StorageItem{
	Link:        "url",
	ShortLink:   "shortUrl",
	ID:          "2",
	DeletedFlag: true,
}

func TestStorage_Init(t *testing.T) {
	type want struct {
		count int
	}

	tests := []struct {
		name string
		want want
	}{
		{
			name: `Test Init`,
			want: want{
				count: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			MemoryStorage.Init()
			length := len(MemoryStorage.data)
			assert.Equal(t, tt.want.count, length)
		})
	}
}

func TestStorage_Set(t *testing.T) {
	type want struct {
		count int
	}
	type args struct {
		data StorageItem
	}

	tests := []struct {
		args args
		name string
		want want
	}{
		{
			name: `Test Set First`,
			args: args{data: TestItem1},
			want: want{
				count: 1,
			},
		},
		{
			name: `Test Set Second`,
			args: args{data: TestItem2},
			want: want{
				count: 2,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			MemoryStorage.Set(tt.args.data)
			length := len(MemoryStorage.data)
			assert.Equal(t, tt.want.count, length)
		})
	}
}

func TestStorage_Get(t *testing.T) {
	type want struct {
		count  int
		result []StorageItem
	}

	tests := []struct {
		name string
		want want
	}{
		{
			name: `Test Get`,
			want: want{
				count:  2,
				result: []StorageItem{TestItem1, TestItem2},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			MemoryStorage.Get()
			length := len(MemoryStorage.data)
			assert.Equal(t, tt.want.count, length)
			assert.Equal(t, tt.want.result, MemoryStorage.data)
		})
	}
}

func TestStorage_Enabled(t *testing.T) {
	type want struct {
		enabled bool
	}

	type args struct {
		enabled bool
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: `Test Enabled Default`,
			want: want{
				enabled: true,
			},
		},
		{
			name: `Test Enabled Change 1`,
			args: args{
				enabled: false,
			},
			want: want{
				enabled: false,
			},
		},
		{
			name: `Test Enabled Change 2`,
			args: args{
				enabled: true,
			},
			want: want{
				enabled: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name != `Test Enabled Default` {
				MemoryStorage.enabled = tt.args.enabled
			}
			assert.Equal(t, tt.want.enabled, MemoryStorage.Enabled())
		})
	}

}

func TestStorage_Clear(t *testing.T) {
	type want struct {
		clearStorage []StorageItem
	}

	tests := []struct {
		name string
		want want
	}{
		{
			name: `Test Clear`,
			want: want{
				clearStorage: make([]StorageItem, 0),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			MemoryStorage.Clear()
			assert.Equal(t, tt.want.clearStorage, MemoryStorage.data)
		})
	}

}
