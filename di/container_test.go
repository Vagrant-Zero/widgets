package di

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewContainer(t *testing.T) {
	tests := []struct {
		name string
		want Container
	}{
		{
			name: "should return a new container with initialized maps and init status",
			want: &DefaultContainer{
				interfaceMap: make(map[string]interface{}),
				typeMap:      make(map[reflect.Type]interface{}),
				status:       initStatus,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewContainer()
			assert.Equal(t, tt.want, got)

			// Verify the concrete type is DefaultContainer
			_, ok := got.(*DefaultContainer)
			assert.True(t, ok)

			// Verify maps are initialized
			container := got.(*DefaultContainer)
			assert.NotNil(t, container.interfaceMap)
			assert.NotNil(t, container.typeMap)
			assert.Equal(t, initStatus, container.status)
		})
	}
}

func TestDefaultContainer_Register(t *testing.T) {
	t.Run("successful registration with name", func(t *testing.T) {
		container := &DefaultContainer{
			interfaceMap: make(map[string]interface{}),
			typeMap:      make(map[reflect.Type]interface{}),
		}
		testImpl := &testInterfaceImpl{}

		container.Register("test", testImpl)

		assert.Equal(t, testImpl, container.interfaceMap["test"])
		assert.Equal(t, testImpl, container.typeMap[reflect.TypeOf(testImpl)])
	})

	t.Run("successful registration without name", func(t *testing.T) {
		container := &DefaultContainer{
			interfaceMap: make(map[string]interface{}),
			typeMap:      make(map[reflect.Type]interface{}),
		}
		testImpl := &testInterfaceImpl{}

		container.Register("", testImpl)

		assert.Empty(t, container.interfaceMap)
		assert.Equal(t, testImpl, container.typeMap[reflect.TypeOf(testImpl)])
	})

	t.Run("panic when impl is nil", func(t *testing.T) {
		container := &DefaultContainer{
			interfaceMap: make(map[string]interface{}),
			typeMap:      make(map[reflect.Type]interface{}),
		}

		assert.PanicsWithValue(t, "interface: test can not be nil or must be a pointer, realType: <nil>", func() {
			container.Register("test", nil)
		})
	})

	t.Run("panic when name already registered", func(t *testing.T) {
		container := &DefaultContainer{
			interfaceMap: make(map[string]interface{}),
			typeMap:      make(map[reflect.Type]interface{}),
		}
		testImpl := &testInterfaceImpl{}
		container.Register("test", testImpl)

		assert.PanicsWithValue(t, "interface already registered: test", func() {
			container.Register("test", testImpl)
		})
	})

	t.Run("panic when impl is not a pointer", func(t *testing.T) {
		container := &DefaultContainer{
			interfaceMap: make(map[string]interface{}),
			typeMap:      make(map[reflect.Type]interface{}),
		}
		testImpl := testInterfaceImpl{}

		assert.PanicsWithValue(t, "interface: test can not be nil or must be a pointer, realType: di.testInterfaceImpl", func() {
			container.Register("test", testImpl)
		})
	})

	t.Run("panic when type already registered", func(t *testing.T) {
		container := &DefaultContainer{
			interfaceMap: make(map[string]interface{}),
			typeMap:      make(map[reflect.Type]interface{}),
		}
		testImpl := &testInterfaceImpl{}
		container.typeMap[reflect.TypeOf(testImpl)] = testImpl

		assert.PanicsWithValue(t, "interface already registered: *di.testInterfaceImpl", func() {
			container.Register("", testImpl)
		})
	})
}

type testInterfaceImpl struct{}

func TestDefaultContainer_TryGet(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *DefaultContainer
		input    string
		expected interface{}
	}{
		{
			name: "empty name returns nil",
			setup: func() *DefaultContainer {
				return &DefaultContainer{
					interfaceMap: make(map[string]interface{}),
				}
			},
			input:    "",
			expected: nil,
		},
		{
			name: "non-existent key returns nil",
			setup: func() *DefaultContainer {
				return &DefaultContainer{
					interfaceMap: make(map[string]interface{}),
				}
			},
			input:    "nonexistent",
			expected: nil,
		},
		{
			name: "existing key returns value",
			setup: func() *DefaultContainer {
				c := &DefaultContainer{
					interfaceMap: make(map[string]interface{}),
				}
				c.interfaceMap["test"] = "test value"
				return c
			},
			input:    "test",
			expected: "test value",
		},
		{
			name: "nil interfaceMap returns nil",
			setup: func() *DefaultContainer {
				return &DefaultContainer{
					interfaceMap: nil,
				}
			},
			input:    "any",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			container := tt.setup()
			result := container.TryGet(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDefaultContainer_MustGet(t *testing.T) {
	tests := []struct {
		name          string
		container     *DefaultContainer
		input         string
		expectedValue interface{}
		expectedError error
	}{
		{
			name: "successful get",
			container: &DefaultContainer{
				interfaceMap: map[string]interface{}{
					"test": "test value",
				},
			},
			input:         "test",
			expectedValue: "test value",
			expectedError: nil,
		},
		{
			name: "empty name",
			container: &DefaultContainer{
				interfaceMap: make(map[string]interface{}),
			},
			input:         "",
			expectedValue: nil,
			expectedError: interfaceNilError,
		},
		{
			name: "name not found",
			container: &DefaultContainer{
				interfaceMap: map[string]interface{}{
					"other": "value",
				},
			},
			input:         "test",
			expectedValue: nil,
			expectedError: interfaceNilError,
		},
		{
			name: "nil interface value",
			container: &DefaultContainer{
				interfaceMap: map[string]interface{}{
					"test": nil,
				},
			},
			input:         "test",
			expectedValue: nil,
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := tt.container.MustGet(tt.input)
			assert.Equal(t, tt.expectedValue, val)
			assert.Equal(t, tt.expectedError, err)
		})
	}
}

func TestDefaultContainer_Initialize(t *testing.T) {
	t.Run("successful initialization", func(t *testing.T) {
		container := &DefaultContainer{
			typeMap: map[reflect.Type]interface{}{
				reflect.TypeOf(&mockService{}): &mockService{},
			},
			status: initStatus,
		}

		assert.NotPanics(t, func() {
			container.Initialize()
		})
		assert.Equal(t, runStatus, container.status)
	})

	t.Run("already initialized", func(t *testing.T) {
		container := &DefaultContainer{
			status: runStatus,
		}

		assert.PanicsWithValue(t, "container already initialized", func() {
			container.Initialize()
		})
	})

	t.Run("nil interface in typeMap", func(t *testing.T) {
		container := &DefaultContainer{
			typeMap: map[reflect.Type]interface{}{
				reflect.TypeOf(&mockService{}): nil,
			},
			status: initStatus,
		}

		assert.PanicsWithValue(t, "interface: *di.mockService is nil or not be a pointer", func() {
			container.Initialize()
		})
	})

	t.Run("non-pointer interface in typeMap", func(t *testing.T) {
		container := &DefaultContainer{
			typeMap: map[reflect.Type]interface{}{
				reflect.TypeOf(mockService{}): mockService{},
			},
			status: initStatus,
		}

		assert.PanicsWithValue(t, "interface: di.mockService is nil or not be a pointer", func() {
			container.Initialize()
		})
	})
}

type mockService struct{}

type Person struct {
	Name string
	Age  int
}

func (p *Person) AfterInject() error {
	fmt.Printf("Person AfterInject called, name: %s, age: %d\n", p.Name, p.Age)
	return nil
}

type School struct {
	TeacherCnt int32
	Teacher    *Person `inject:"person"`
	Student    *Person
}

func (s *School) AfterInject() error {
	fmt.Printf("School AfterInject called, name: %s, age: %d\n", s.Teacher.Name, s.Teacher.Age)
	return nil
}

type UserService struct {
	Workplace string
	Person    *Person `inject:"person"`
}

type CircleA struct {
	B *CircleB `inject:"circleB"`
}

type CircleB struct {
	A *CircleA `inject:"circleA"`
}

func TestDefaultContainer_Initialize2(t *testing.T) {
	container := NewContainer()
	t.Run("successful inject", func(t *testing.T) {
		// 1. 注册组件
		container.Register("person", &Person{Name: "Alice", Age: 20})
		container.Register("school", &School{})
		container.Initialize()

		val, err := container.MustGet("person")
		assert.Nil(t, err)
		assert.Equal(t, &Person{Name: "Alice", Age: 20}, val)

		s, err := container.MustGet("school")
		assert.Nil(t, err)
		assert.Equal(t, &School{
			Teacher: &Person{Name: "Alice", Age: 20},
			Student: &Person{},
		}, s)
		container.Clear()
	})

	t.Run("A depends on B and C depends on B", func(t *testing.T) {
		// 1. 注册组件
		container.Register("person", &Person{Name: "Alice", Age: 20})
		container.Register("school", &School{})
		container.Register("userService", &UserService{})
		container.Initialize()

		val, err := container.MustGet("userService")
		assert.Nil(t, err)
		assert.Equal(t, &UserService{
			Person: &Person{Name: "Alice", Age: 20},
		}, val)

		s, err := container.MustGet("school")
		assert.Nil(t, err)
		assert.Equal(t, &School{
			Teacher: &Person{Name: "Alice", Age: 20},
			Student: &Person{},
		}, s)
		container.Clear()
	})

	//t.Run("cycle dependency", func(t *testing.T) {
	//	// 1. 注册组件
	//	container.Register("circleA", &CircleA{})
	//	container.Register("circleB", &CircleB{})
	//	container.Initialize()
	//
	//	assert.PanicsWithValue(t, "cycle dependency", func() {
	//		container.Initialize()
	//	})
	//})
}
