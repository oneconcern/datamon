package app

import (
	"sync"

	"github.com/kardianos/osext"
	"github.com/spf13/viper"
)

type errString string

func (e errString) Error() string {
	return string(e)
}

const (
	// ErrModuleUnknown returned when no module can be found for the specified key
	ErrModuleUnknown errString = "unknown module"
)

var (
	execName func() (string, error)
)

func init() {
	execName = osext.Executable
}

// A Key represents a key for a module.
// Users of this package can define their own keys, this is just the type definition.
type Key int

// Application is an application level context package
// It can be used as a kind of dependency injection container
type Application interface {
	// Add modules to the application context
	Add(...Module) error

	// Get the module at the specified key, thread-safe
	Get(Key) interface{}

	// Get the module at the specified key, thread-safe
	GetOK(Key) (interface{}, bool)

	// Set the module at the specified key, this should be safe across multiple threads
	Set(Key, interface{}) error

	// Init the application and its modules with the config.
	Init() error

	// Start the application an its enabled modules
	Start() error

	// Stop the application an its enabled modules
	Stop() error
}

// LifecycleCallback function definition
type LifecycleCallback interface {
	Call(Application) error
}

// Init is an initializer for an initialization function
type Init func(Application) error

// Call implements the callback interface
func (fn Init) Call(app Application) error {
	return fn(app)
}

// Start is an initializer for a start function
type Start func(Application) error

// Call implements the callback interface
func (fn Start) Call(app Application) error {
	return fn(app)
}

// Stop is an initializer for a stop function
type Stop func(Application) error

// Call implements the callback interface
func (fn Stop) Call(app Application) error {
	return fn(app)
}

// Reload is an initalizater for a reload function
type Reload func(Application) error

// Call implements the callback interface
func (fn Reload) Call(app Application) error {
	return fn(app)
}

// A Module is a component that has a specific lifecycle
type Module interface {
	Init(Application) error
	Start(Application) error
	Stop(Application) error
	Reload(Application) error
}

// MakeModule by passing the callback functions.
// You can pass multiple callback functions of the same type if you want
func MakeModule(callbacks ...LifecycleCallback) Module {
	var (
		init   []Init
		start  []Start
		reload []Reload
		stop   []Stop
	)

	for _, callback := range callbacks {
		switch cb := callback.(type) {
		case Init:
			init = append(init, cb)
		case Start:
			start = append(start, cb)
		case Stop:
			stop = append(stop, cb)
		case Reload:
			reload = append(reload, cb)
		}
	}

	return &dynamicModule{
		init:   init,
		start:  start,
		reload: reload,
		stop:   stop,
	}
}

type dynamicModule struct {
	init   []Init
	start  []Start
	stop   []Stop
	reload []Reload
}

func (d *dynamicModule) Init(app Application) error {
	for _, cb := range d.init {
		if err := cb.Call(app); err != nil {
			return err
		}
	}
	return nil
}

func (d *dynamicModule) Start(app Application) error {
	for _, cb := range d.start {
		if err := cb.Call(app); err != nil {
			return err
		}
	}
	return nil
}

func (d *dynamicModule) Stop(app Application) error {
	for _, cb := range d.stop {
		if err := cb.Call(app); err != nil {
			return err
		}
	}
	return nil
}

func (d *dynamicModule) Reload(app Application) error {
	for _, cb := range d.reload {
		if err := cb.Call(app); err != nil {
			return err
		}
	}
	return nil
}

// New creates an application context
func New(config *viper.Viper) Application {
	return &defaultApplication{conf: config}
}

type defaultApplication struct {
	modules []Module

	registry sync.Map
	conf     *viper.Viper
}

func (d *defaultApplication) Add(modules ...Module) error {
	d.modules = append(d.modules, modules...)
	return nil
}

// Get the module at the specified key, return nil when the component doesn't exist
func (d *defaultApplication) Get(key Key) interface{} {
	mod, _ := d.GetOK(key)
	return mod
}

// Get the module at the specified key, return false when the component doesn't exist
func (d *defaultApplication) GetOK(key Key) (interface{}, bool) {
	mod, ok := d.registry.Load(key)
	if !ok {
		return nil, ok
	}
	return mod, ok
}

func (d *defaultApplication) Set(key Key, module interface{}) error {
	d.registry.Store(key, module)
	return nil
}

func (d *defaultApplication) Init() error {
	for _, mod := range d.modules {
		if err := mod.Init(d); err != nil {
			return err
		}
	}
	return nil
}

func (d *defaultApplication) Start() error {
	for _, mod := range d.modules {
		if err := mod.Start(d); err != nil {
			return err
		}
	}
	return nil
}

func (d *defaultApplication) Stop() error {
	for _, mod := range d.modules {
		if err := mod.Stop(d); err != nil {
			return err
		}
	}
	return nil
}
