/*
Package di provides a simple dependency injection framework for Go.
This package provide sdk for dependency injection in go.

Sample:
package main

import (
	"fmt"
	"github.com/vagrant-Zero/widgets/di"
)

func main() {
	// 1. create a container
	container := di.NewContainer()

	// 2. register components
	container.Register("userService", &UserService{})

	// 3. initialize all registered components
	container.Initialize()

	// 4. get components
	userService, err := container.MustGet("userService")
	if err != nil {
		panic(err)
	}
	fmt.Println(userService.(*UserService))
}
*/

package di

import (
	"fmt"
	"reflect"
)

type DefaultContainer struct {
	interfaceMap map[string]interface{}
	typeMap      map[reflect.Type]interface{}
	status       int
}

func NewContainer() Container {
	return &DefaultContainer{
		interfaceMap: make(map[string]interface{}),
		typeMap:      make(map[reflect.Type]interface{}),
		status:       initStatus,
	}
}

func (d *DefaultContainer) Register(name string, impl interface{}) {
	if d.status != initStatus {
		panic("container is not initStatus, can not register")
	}
	ty := reflect.TypeOf(impl)
	if ty == nil || ty.Kind() != reflect.Ptr {
		panic(fmt.Sprintf("interface: %s can not be nil or must be a pointer, realType: %v", name, ty))
	}
	if impl == nil || reflect.ValueOf(impl).IsNil() {
		panic("interface is nil")
	}
	if _, ok := d.interfaceMap[name]; ok {
		panic("interface already registered: " + name)
	}
	if _, ok := d.typeMap[ty]; ok {
		panic("interface already registered: " + ty.String())
	}
	if name != "" {
		d.interfaceMap[name] = impl
	}
	d.typeMap[ty] = impl
}

func (d *DefaultContainer) TryGet(name string) interface{} {
	if name == "" {
		return nil
	}
	return d.interfaceMap[name]
}

func (d *DefaultContainer) MustGet(name string) (interface{}, error) {
	if name == "" {
		return nil, interfaceNilError
	}
	val, ok := d.interfaceMap[name]
	if !ok {
		return nil, interfaceNilError
	}
	return val, nil
}

func (d *DefaultContainer) Initialize() {
	// 1. 检查状态，如果已经初始化过，panic
	if d.status == runStatus {
		panic("container already initialized")
	}

	// 2. 遍历interfaceMap，初始化每个interface
	registeredMap := make(map[reflect.Type]interface{}) // 全局的map，用于记录已经初始化过的val
	for t, val := range d.typeMap {
		// 2.1 参数校验
		t1 := reflect.TypeOf(val)
		if t1 == nil || t1.Kind() != reflect.Ptr {
			panic("interface: " + t.String() + " is nil or not be a pointer")
		}

		// 2.2 处理每个interface的字段，完成注入
		err := d.processInterface(val, registeredMap, make(map[reflect.Type]struct{}))
		if err != nil {
			panic(err)
		}

	}

	// 3. 修改状态为已初始化
	d.status = runStatus
}

func (d *DefaultContainer) Clear() {
	d.status = initStatus
	d.interfaceMap = make(map[string]interface{})
	d.typeMap = make(map[reflect.Type]interface{})
}

func (d *DefaultContainer) processInterface(v interface{},
	registeredMap map[reflect.Type]interface{},
	registeredForInterfaceSet map[reflect.Type]struct{}) error {

	if reflect.TypeOf(v).Kind() != reflect.Ptr {
		return fmt.Errorf("interface: %s must be a pointer", reflect.TypeOf(v).Kind())
	}
	// 0. registeredMap 全局的map，用于记录已经初始化过的val
	// 如果已经初始化，直接返回
	if _, ok := registeredMap[reflect.TypeOf(v)]; ok {
		return nil
	}

	// 0.1 第一次进入的时候，registeredForInterfaceMap为空map，由最外层传入，此map只用于记录当前interface下，是否出现循环依赖，如果出现循环依赖，panic
	if _, ok := registeredForInterfaceSet[reflect.TypeOf(v)]; ok {
		panic("cycle dependency" /*+ reflect.TypeOf(v).String()*/)
	}
	// 当前v正在inject中
	registeredForInterfaceSet[reflect.TypeOf(v)] = struct{}{}

	// 1. 为v的每个字段注入依赖，v是一个指针类型，先拿到指针对应的值
	vv := reflect.ValueOf(v)
	val := vv.Elem()
	typ := val.Type()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		// 1.1 如果依赖有inject tag（inject tag即interface的name），则递归注入（不支持循环依赖）， 完成后，从registeredMap中获取该值，然后赋值给字段
		tag := field.Tag.Get(injectTag)
		if tag == "" {
			// 1.2 如果依赖没有inject tag，并且该值是指针类型，且是exported（即首字母大写），则注入该值的零值（如果是指针对象，创建空的指针对象，而不是nil），然后赋值给字段
			if field.Type.Kind() == reflect.Ptr {
				if field.Type.Elem().Kind() == reflect.Struct {
					// 如果是struct类型，创建一个空的struct对象，然后赋值给字段
					reflect.ValueOf(v).Elem().Field(i).Set(reflect.New(field.Type.Elem()))
				}
			}
			continue
		}
		if field.Type.Kind() != reflect.Ptr {
			return fmt.Errorf("interface: %s must be a pointer", field.Type.String())
		}
		if !field.IsExported() {
			return fmt.Errorf("interface: %s is not exported", field.Type.String())
		}
		// 1.3 如果依赖有inject tag，则递归注入（不支持循环依赖）， 完成后，从registeredMap中获取该值，然后赋值给字段
		childField, ok := d.interfaceMap[tag]
		if !ok {
			return fmt.Errorf("interface with tag not registered: %s", tag)
		}
		if r, ok := registeredMap[field.Type]; ok {
			reflect.ValueOf(v).Elem().Field(i).Set(reflect.ValueOf(r))
			continue
		}
		// 1.4 如果依赖没有注入过，递归注入
		err := d.processInterface(childField, registeredMap, registeredForInterfaceSet)
		if err != nil {
			return err
		}
		// 1.5 从registeredMap中获取该值，然后赋值给字段
		vv.Elem().Field(i).Set(reflect.ValueOf(registeredMap[field.Type]))
	}

	// 2. 当前val的所有字段都已经注入完毕，将val加入到registeredMap中
	registeredMap[reflect.TypeOf(v)] = v

	// 3. 如果val是injector，完成该injector的所有字段注入后，调用AfterInject方法
	injector, ok := v.(Injector)
	if !ok {
		return nil
	}
	return injector.AfterInject()
}
