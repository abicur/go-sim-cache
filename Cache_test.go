package cache

import (
	"context"
	"math"
	"reflect"
	"testing"
	"time"
)

func TestNewInMemoryCache(t *testing.T) {
	tests := []struct {
		name    string
		options []func(cache *inMemoryCache)
	}{
		{
			name: "New in memory cache",
		},
		{
			name: "New in memory cache with options",
			options: []func(cache *inMemoryCache){
				WithCleanUpInterval(time.Millisecond * 20),
			},
		},
	}
	ctx := context.Background()
	var cache Cache
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache = NewInMemoryCache(ctx, tt.options...)
			_, ok := cache.(*inMemoryCache)
			if !ok {
				t.Errorf("Invalid constructor result. Expect *inMemoryCache")
			}
		})
	}
}

func TestWithCleanUpInterval(t *testing.T) {
	expectedDuration := time.Millisecond * 100
	cache := &inMemoryCache{cleanUpTicker: time.NewTicker(time.Second * 5)}

	fn := WithCleanUpInterval(expectedDuration)
	fn(cache)

	beginTime := time.Now()
	<-cache.cleanUpTicker.C
	endTime := time.Now()

	actualDuration := endTime.Sub(beginTime)
	actualDifference := math.Round(float64(actualDuration.Milliseconds()) / 100)
	expectedDifference := math.Round(float64(expectedDuration.Milliseconds()) / 100)

	if actualDifference != expectedDifference {
		t.Errorf(
			"WithCleanUpInterval() unexpected ticker duration. Expected %d, but actual %d",
			expectedDuration.Milliseconds(),
			actualDuration.Milliseconds(),
		)
	}
}

func Test_inMemoryCache_Delete(t *testing.T) {
	type args struct {
		key             string
		isValueExisting bool
		value           interface{}
		expiredInterval time.Duration
	}

	tests := []struct {
		name string
		args args
	}{
		{
			"Delete item",
			args{
				key:             "test",
				isValueExisting: true,
				expiredInterval: time.Second * 10,
			},
		},
		{
			"Delete a non-existent value",
			args{
				key:             "test",
				isValueExisting: false,
			},
		},
	}
	var cache Cache
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache = &inMemoryCache{}
			if tt.args.isValueExisting {
				cache.Set(tt.args.key, 42, tt.args.expiredInterval)
				value, exist := cache.Get(tt.args.key)
				intValue, ok := value.(int)
				if !exist || !ok || intValue != 42 {
					t.Errorf("Setting value failed")
				}
			}
			cache.Delete(tt.args.key)
			_, exist := cache.Get(tt.args.key)
			if exist {
				t.Errorf("Delete() method didn't delete item with key %s", tt.args.key)
			}
		})
	}
}

func Test_inMemoryCache_Get(t *testing.T) {
	type args struct {
		key             string
		isValueExisting bool
		value           interface{}
		expiredInterval time.Duration
	}
	tests := []struct {
		name              string
		args              args
		expectedValue     interface{}
		expectedExistence bool
	}{
		{
			name: "Get value from cache",
			args: args{
				key:             "test",
				isValueExisting: true,
				value:           42,
				expiredInterval: time.Second * 20,
			},
			expectedValue:     42,
			expectedExistence: true,
		},
		{
			name: "Get a non-existent value",
			args: args{
				key:             "test",
				isValueExisting: false,
			},
			expectedValue:     nil,
			expectedExistence: false,
		},
		{
			name: "Get expired value",
			args: args{
				key:             "test",
				isValueExisting: true,
				value:           42,
				expiredInterval: 0,
			},
			expectedValue:     nil,
			expectedExistence: false,
		},
	}

	var cache Cache
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache = &inMemoryCache{}

			if tt.args.isValueExisting {
				cache.Set(tt.args.key, tt.args.value, tt.args.expiredInterval)
			}

			actualValue, actualExistence := cache.Get(tt.args.key)
			if !reflect.DeepEqual(actualValue, tt.expectedValue) {
				t.Errorf("Get() actualValue = %v, want %v", actualValue, tt.expectedValue)
			}
			if actualExistence != tt.expectedExistence {
				t.Errorf("Get() actualExistence = %v, want %v", actualExistence, tt.expectedExistence)
			}
		})
	}
}

func Test_inMemoryCache_Set(t *testing.T) {
	type args struct {
		key                 string
		value               interface{}
		expiredInterval     time.Duration
		needToSetValueTwice bool
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Set simple value",
			args: args{
				key:                 "test",
				value:               42,
				expiredInterval:     time.Second * 10,
				needToSetValueTwice: false,
			},
		},
		{
			name: "Set simple value",
			args: args{
				key:                 "test",
				value:               2,
				expiredInterval:     time.Second * 10,
				needToSetValueTwice: true,
			},
		},
		{
			name: "Set object value",
			args: args{
				key: "test",
				value: struct {
					value int
				}{
					value: 42,
				},
				expiredInterval:     time.Second * 10,
				needToSetValueTwice: true,
			},
		},
	}

	var cache Cache
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache = &inMemoryCache{}
			cache.Set(tt.args.key, tt.args.value, tt.args.expiredInterval)

			value, actualExistence := cache.Get(tt.args.key)
			if !reflect.DeepEqual(value, tt.args.value) {
				t.Errorf("Get() got = %v, want %v", value, tt.args.value)
			}
			if actualExistence != true {
				t.Errorf("Get() actualExistence = %v, want %v", actualExistence, true)
			}

			if tt.args.needToSetValueTwice {
				cache.Set(tt.args.key, tt.args.value, tt.args.expiredInterval)
				if !reflect.DeepEqual(value, tt.args.value) {
					t.Errorf("Get() got = %v, want %v", value, tt.args.value)
				}
				if actualExistence != true {
					t.Errorf("Get() actualExistence = %v, want %v", actualExistence, true)
				}
			}
		})
	}
}

func Test_inMemoryCache_cleanCache(t *testing.T) {
	tests := []struct {
		name  string
		items []struct {
			key      string
			value    interface{}
			interval time.Duration
		}
		want []interface{}
	}{
		{
			name: "Find cache items to delete",
			items: []struct {
				key      string
				value    interface{}
				interval time.Duration
			}{
				{
					key:      "test1",
					value:    42,
					interval: time.Second * 5,
				},
				{
					key:      "test2",
					value:    43,
					interval: time.Millisecond,
				},
				{
					key:      "test3",
					value:    44,
					interval: 0,
				},
				{
					key:      "test4",
					value:    45,
					interval: time.Millisecond * 30,
				},
			},
			want: []interface{}{
				"test1",
				"test4",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ticker := time.NewTicker(time.Millisecond * 20)
			cache := &inMemoryCache{
				cleanUpTicker: ticker,
			}
			for _, item := range tt.items {
				cache.Set(item.key, item.value, item.interval)
			}

			ctx, cancelFn := context.WithTimeout(context.Background(), time.Millisecond*25)
			cache.cleanUpCache(ctx)
			cancelFn()

			cache.storage.Range(func(key, value interface{}) bool {
				result := false
				for _, wantKey := range tt.want {
					if wantKey == key {
						result = true
						break
					}
				}
				if result != true {
					t.Errorf("cleanUpCache() clean result unexpected. Expect to see key = %s in storage", key)
				}

				return result
			})
		})
	}
}

func Test_inMemoryCache_getCacheItemsToDelete(t *testing.T) {
	tests := []struct {
		name  string
		items []struct {
			key      string
			value    interface{}
			interval time.Duration
		}
		want []interface{}
	}{
		{
			name: "Find cache items to delete",
			items: []struct {
				key      string
				value    interface{}
				interval time.Duration
			}{
				{
					key:      "test1",
					value:    42,
					interval: time.Second * 5,
				},
				{
					key:      "test2",
					value:    43,
					interval: time.Millisecond,
				},
				{
					key:      "test3",
					value:    44,
					interval: 0,
				},
			},
			want: []interface{}{
				"test2",
				"test3",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := &inMemoryCache{}
			for _, item := range tt.items {
				cache.Set(item.key, item.value, item.interval)
			}
			time.Sleep(time.Millisecond)
			got := cache.getCacheItemsToDelete()

			if len(got) != len(tt.want) {
				t.Errorf("getCacheItemsToDelete() an unexpected result. want %v, got %v", tt.want, got)
			}

			for _, wantItem := range tt.want {
				itemFound := false
				for _, itemToDelete := range got {
					if itemToDelete == wantItem {
						itemFound = true
						break
					}
				}

				if itemFound == false {
					t.Errorf("getCacheItemsToDelete() an unexpected result. want %v, got %v", tt.want, got)
				}
			}
		})
	}
}
