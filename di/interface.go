package di

type Injector interface {
	// AfterInject will be called after this component is initialized.
	AfterInject() error
}

type Container interface {
	// Register a component.
	// The name is optional, if name is existed, container will panic.
	// The impl must be a pointer.
	// same type can only be registered once.
	Register(name string, impl interface{})
	// TryGet a component by name.
	// If the component is not registered, return nil.
	TryGet(name string) interface{}
	// MustGet a component by name.
	// If the component is not registered, return error.
	MustGet(name string) (interface{}, error)
	// Initialize all registered components.
	// This method is not thread-safe, expected to be called only once.
	Initialize()
	// Clear all registered components.
	Clear()
}
