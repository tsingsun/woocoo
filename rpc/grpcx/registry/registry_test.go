package registry

import (
	"encoding/base64"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/resolver"
	"net/url"
	"strings"
	"testing"
)

func TestTargetToOptions(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		optionsJSON := `{"Namespace":"test", "ServiceName":"service"}`
		encoded := base64.URLEncoding.EncodeToString([]byte(optionsJSON))
		target := resolver.Target{
			URL: url.URL{
				RawQuery: "options=" + encoded,
				Host:     "host.com",
				Path:     "path",
				Opaque:   "opaque",
			},
		}

		gotOptions, err := TargetToOptions(target)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if gotOptions.Namespace != "test" || gotOptions.ServiceName != "service" {
			t.Errorf("expected Namespace: 'test', ServiceName: 'service', got: %+v", gotOptions)
		}
	})

	t.Run("InvalidBase64", func(t *testing.T) {
		target := resolver.Target{
			URL: url.URL{
				RawQuery: "options=invalidBase64",
			},
		}

		_, err := TargetToOptions(target)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "fail to decode") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		encoded := base64.URLEncoding.EncodeToString([]byte(`{"invalid`)) // 不完整的 JSON
		target := resolver.Target{
			URL: url.URL{
				RawQuery: "options=" + encoded,
			},
		}

		_, err := TargetToOptions(target)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "fail to unmarshal") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("NoOptionKey", func(t *testing.T) {
		target := resolver.Target{
			URL: url.URL{
				RawQuery: "anotherKey=value",
			},
		}

		options, err := TargetToOptions(target)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if options.Namespace != "" || options.ServiceName != "" {
			t.Errorf("expected empty fields, got %+v", options)
		}
	})

	t.Run("EmptyOptionsStr", func(t *testing.T) {
		target := resolver.Target{
			URL: url.URL{
				RawQuery: "options=",
			},
		}

		options, err := TargetToOptions(target)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if options.Namespace != "" || options.ServiceName != "" {
			t.Errorf("expected empty fields, got %+v", options)
		}
	})

	t.Run("NoQueryParamsWithOpaque", func(t *testing.T) {
		target := resolver.Target{
			URL: url.URL{
				Host:   "namespace.com",
				Opaque: "service-opaque",
			},
		}

		options, err := TargetToOptions(target)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if options.Namespace != "namespace.com" || options.ServiceName != "service-opaque" {
			t.Errorf("expected Namespace: 'namespace.com', ServiceName: 'service-opaque', got %+v", options)
		}
	})

	t.Run("NoQueryParamsWithoutOpaque", func(t *testing.T) {
		target := resolver.Target{
			URL: url.URL{
				Host: "namespace.com",
				Path: "/service-path",
			},
		}

		options, err := TargetToOptions(target)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if options.Namespace != "namespace.com" || options.ServiceName != "/service-path" {
			t.Errorf("expected ServiceName: '/service-path', got %+v", options)
		}
	})

	t.Run("MultipleOptionValues", func(t *testing.T) {
		optionsJSON := `{"Namespace":"test"}`
		encoded := base64.URLEncoding.EncodeToString([]byte(optionsJSON))
		target := resolver.Target{
			URL: url.URL{
				RawQuery: "options=" + encoded + "&options=another",
			},
		}

		options, err := TargetToOptions(target)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if options.Namespace != "test" {
			t.Errorf("expected Namespace 'test', got '%s'", options.Namespace)
		}
	})
}

func TestServiceInfo_ToAttributes(t *testing.T) {
	t.Run("TestEmptyMetadata", func(t *testing.T) {
		// 准备数据
		serviceInfo := ServiceInfo{
			Metadata: map[string]string{},
		}
		result := serviceInfo.ToAttributes()
		assert.Nil(t, result)
	})
	t.Run("TestSingleKeyValuePair", func(t *testing.T) {
		serviceInfo := ServiceInfo{
			Metadata: map[string]string{
				"key1": "value1",
			},
		}
		result := serviceInfo.ToAttributes()
		assert.Equal(t, "value1", result.Value("key1"))
	})
	t.Run("TestMultipleKeyValues", func(t *testing.T) {
		serviceInfo := ServiceInfo{
			Metadata: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		}
		result := serviceInfo.ToAttributes()
		assert.Equal(t, "value1", result.Value("key1"))
		assert.Equal(t, "value2", result.Value("key2"))
	})
}

func TestAddress_Normal(t *testing.T) {
	testCases := []struct {
		name     string
		service  ServiceInfo
		expected string
	}{
		{
			name:     "normal",
			service:  ServiceInfo{Host: "localhost", Port: 8080},
			expected: "localhost:8080",
		},
		{
			name:     "empty host",
			service:  ServiceInfo{Host: "", Port: 80},
			expected: ":80",
		},
		{
			name:     "port 0",
			service:  ServiceInfo{Host: "localhost", Port: 0},
			expected: "localhost:0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.service.Address()
			if got != tc.expected {
				t.Errorf("Address() = %v, want %v", got, tc.expected)
			}
		})
	}
}

func TestBuildKey(t *testing.T) {
	testCases := []struct {
		name     string
		service  ServiceInfo
		expected string
	}{
		{
			name: "normal",
			service: ServiceInfo{
				Namespace: "ns",
				Name:      "service",
				Version:   "1",
				Host:      "127.0.0.1",
				Port:      8080,
			},
			expected: "/ns/service/1/127.0.0.1:8080",
		},
		{
			name: "Namespace empty",
			service: ServiceInfo{
				Namespace: "",
				Name:      "service",
				Version:   "1",
				Host:      "127.0.0.1",
				Port:      8080,
			},
			expected: "//service/1/127.0.0.1:8080",
		},
		{
			name: "Name empty",
			service: ServiceInfo{
				Namespace: "ns",
				Name:      "",
				Version:   "1",
				Host:      "127.0.0.1",
				Port:      8080,
			},
			expected: "/ns//1/127.0.0.1:8080",
		},
		{
			name: "Version empty",
			service: ServiceInfo{
				Namespace: "ns",
				Name:      "service",
				Version:   "",
				Host:      "127.0.0.1",
				Port:      8080,
			},
			expected: "/ns/service//127.0.0.1:8080",
		},
		{
			name: "Address empty",
			service: ServiceInfo{
				Namespace: "ns",
				Name:      "service",
				Version:   "1",
			},
			expected: "/ns/service/1/:0",
		},
		{
			name: "empty",
			service: ServiceInfo{
				Namespace: "",
				Name:      "",
			},
			expected: "////:0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.service.BuildKey()
			if actual != tc.expected {
				t.Errorf("Expected: %q, got: %q", tc.expected, actual)
			}
		})
	}
}

type MockDriver struct {
	Driver
	name string
	tag  string
}

func (m MockDriver) Name() string {
	return m.name
}

// 测试 RegisterDriver 函数
func TestRegisterDriver(t *testing.T) {
	t.Run("Register new scheme", func(t *testing.T) {
		scheme := "mock"
		mockDrv := MockDriver{name: "mock"}
		RegisterDriver(scheme, mockDrv)

		assert.NotNil(t, driverManager[scheme])
	})

	t.Run("Overwrite existing scheme", func(t *testing.T) {
		scheme := "mock1"
		mockDrv1 := MockDriver{name: "mock1"}
		mockDrv2 := MockDriver{name: "mock1", tag: "mt"}

		RegisterDriver(scheme, mockDrv1)
		RegisterDriver(scheme, mockDrv2)
		got, _ := GetRegistry(scheme)
		assert.Equal(t, mockDrv2.tag, got.(MockDriver).tag)
	})
}
