# DI

- provides a simple dependency injection framework for Go
- just support pointer type
- not support circular dependency

## Usage

```go
package main

import (
	"fmt"
	"github.com/vagrant-Zero/widgets/di"
)

type Person struct {
	Name string
	Age  int
}

// AfterInject do something after injection
func (p *Person) AfterInject() error {
	fmt.Printf("Person AfterInject called, name: %s, age: %d\n", p.Name, p.Age)
	return nil
}

type School struct {
	TeacherCnt int32
	Teacher    *Person `inject:"person"`
	Student    *Person
}

// AfterInject do something after injection
func (s *School) AfterInject() error {
	fmt.Printf("School AfterInject called, name: %s, age: %d\n", s.Teacher.Name, s.Teacher.Age)
	s.TeacherCnt = 2
	return nil
}

type UserService struct {
	Workplace string
	Person    *Person `inject:"person"`
}

type NoTagService struct {
	Person *Person `inject:""` // no tag, will use the field's type inject
}

func main() {
	// 1. create a container
	container := di.NewContainer()

	// 2. register components
	container.Register("person", &Person{Name: "Alice", Age: 20})
	container.Register("school", &School{})
	container.Register("userService", &UserService{})
	container.Register("noTagService", &NoTagService{})

	// 3. initialize all registered components
	container.Initialize()

	// 4. get components
	val, err := container.MustGet("userService")
	if err != nil {
		panic(err)
	}
	fmt.Printf("userService: %+v\n", val)
}
```
