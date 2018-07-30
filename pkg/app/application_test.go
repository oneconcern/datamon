package app

import (
	"errors"

	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApplication_AddModule(t *testing.T) {
	var initCount, startCount, stopCount, reloadCount int
	var successMod = MakeModule(
		Init(func(_ Application) error {
			initCount++
			return nil
		}),
		Init(func(_ Application) error {
			initCount++
			return nil
		}),
		Start(func(_ Application) error {
			startCount++
			return nil
		}),
		Start(func(_ Application) error {
			startCount++
			return nil
		}),
		Start(func(_ Application) error {
			startCount++
			return nil
		}),
		Reload(func(_ Application) error {
			reloadCount++
			return nil
		}),
		Stop(func(_ Application) error {
			stopCount++
			return nil
		}),
		Stop(func(_ Application) error {
			stopCount++
			return nil
		}),
		Stop(func(_ Application) error {
			stopCount++
			return nil
		}),
		Stop(func(_ Application) error {
			stopCount++
			return nil
		}),
		Stop(func(_ Application) error {
			stopCount++
			return nil
		}),
	)

	var otherMod = MakeModule(
		Init(func(_ Application) error {
			initCount++
			return nil
		}),
		Init(func(_ Application) error {
			initCount++
			return nil
		}),
		Start(func(_ Application) error {
			startCount++
			return nil
		}),
		Start(func(_ Application) error {
			startCount++
			return nil
		}),
		Start(func(_ Application) error {
			startCount++
			return nil
		}),
		Reload(func(_ Application) error {
			reloadCount++
			return nil
		}),
		Stop(func(_ Application) error {
			stopCount++
			return nil
		}),
		Stop(func(_ Application) error {
			stopCount++
			return nil
		}),
		Stop(func(_ Application) error {
			stopCount++
			return nil
		}),
		Stop(func(_ Application) error {
			stopCount++
			return nil
		}),
		Stop(func(_ Application) error {
			stopCount++
			return nil
		}),
	)

	app := New(nil)
	app.Add(successMod, otherMod)
	assert.Len(t, app.(*defaultApplication).modules, 2)

	if assert.NoError(t, app.Init()) {
		assert.Equal(t, initCount, 4)
	}

	if assert.NoError(t, app.Start()) {
		assert.Equal(t, startCount, 6)
	}

	if assert.NoError(t, app.Stop()) {
		assert.Equal(t, stopCount, 10)
	}
}

func TestApplication_AddModuleError(t *testing.T) {
	var initCount, startCount, stopCount, reloadCount int
	var successMod = MakeModule(
		Init(func(_ Application) error {
			initCount++
			return nil
		}),
		Init(func(_ Application) error {
			initCount++
			return nil
		}),
		Start(func(_ Application) error {
			startCount++
			return nil
		}),
		Start(func(_ Application) error {
			startCount++
			return nil
		}),
		Start(func(_ Application) error {
			startCount++
			return nil
		}),
		Reload(func(_ Application) error {
			reloadCount++
			return nil
		}),
		Stop(func(_ Application) error {
			stopCount++
			return nil
		}),
		Stop(func(_ Application) error {
			stopCount++
			return nil
		}),
		Stop(func(_ Application) error {
			stopCount++
			return nil
		}),
		Stop(func(_ Application) error {
			stopCount++
			return nil
		}),
		Stop(func(_ Application) error {
			stopCount++
			return nil
		}),
	)

	var failMod = MakeModule(
		Init(func(_ Application) error {
			initCount++
			return errors.New("expected")
		}),
		Init(func(_ Application) error {
			initCount++
			return nil
		}),
		Start(func(_ Application) error {
			startCount++
			return nil
		}),
		Start(func(_ Application) error {
			startCount++
			return errors.New("expected")
		}),
		Start(func(_ Application) error {
			startCount++
			return nil
		}),
		Reload(func(_ Application) error {
			reloadCount++
			return errors.New("expected")
		}),
		Stop(func(_ Application) error {
			stopCount++
			return nil
		}),
		Stop(func(_ Application) error {
			stopCount++
			return errors.New("expected")
		}),
		Stop(func(_ Application) error {
			stopCount++
			return nil
		}),
		Stop(func(_ Application) error {
			stopCount++
			return nil
		}),
		Stop(func(_ Application) error {
			stopCount++
			return nil
		}),
	)

	app := New(nil)
	app.Add(successMod, failMod)
	assert.Len(t, app.(*defaultApplication).modules, 2)

	if assert.Error(t, app.Init()) {
		assert.Equal(t, initCount, 3)
	}

	if assert.Error(t, app.Start()) {
		assert.Equal(t, startCount, 5)
	}

	if assert.Error(t, app.Stop()) {
		assert.Equal(t, stopCount, 7)
	}
}

func TestApplication_MakeModule(t *testing.T) {
	var initCount, startCount, stopCount, reloadCount int
	var mod = MakeModule(
		Init(func(_ Application) error {
			initCount++
			return nil
		}),
		Init(func(_ Application) error {
			initCount++
			return nil
		}),
		Start(func(_ Application) error {
			startCount++
			return nil
		}),
		Start(func(_ Application) error {
			startCount++
			return nil
		}),
		Start(func(_ Application) error {
			startCount++
			return nil
		}),
		Reload(func(_ Application) error {
			reloadCount++
			return nil
		}),
		Stop(func(_ Application) error {
			stopCount++
			return nil
		}),
		Stop(func(_ Application) error {
			stopCount++
			return nil
		}),
		Stop(func(_ Application) error {
			stopCount++
			return nil
		}),
		Stop(func(_ Application) error {
			stopCount++
			return nil
		}),
		Stop(func(_ Application) error {
			stopCount++
			return nil
		}),
	).(*dynamicModule)

	assert.Len(t, mod.init, 2)
	assert.Len(t, mod.start, 3)
	assert.Len(t, mod.stop, 5)
	assert.Len(t, mod.reload, 1)

	assert.NoError(t, mod.Init(nil))
	assert.NoError(t, mod.Start(nil))
	assert.NoError(t, mod.Stop(nil))
	assert.NoError(t, mod.Reload(nil))

	assert.Equal(t, 2, initCount)
	assert.Equal(t, 3, startCount)
	assert.Equal(t, 1, reloadCount)
	assert.Equal(t, 5, stopCount)
}

func TestApplication_MakeModuleError(t *testing.T) {
	var initCount, startCount, stopCount, reloadCount int
	var mod = MakeModule(
		Init(func(_ Application) error {
			initCount++
			return errors.New("expected")
		}),
		Init(func(_ Application) error {
			initCount++
			return nil
		}),
		Start(func(_ Application) error {
			startCount++
			return nil
		}),
		Start(func(_ Application) error {
			startCount++
			return errors.New("expected")
		}),
		Start(func(_ Application) error {
			startCount++
			return nil
		}),
		Reload(func(_ Application) error {
			reloadCount++
			return errors.New("expected")
		}),
		Stop(func(_ Application) error {
			stopCount++
			return nil
		}),
		Stop(func(_ Application) error {
			stopCount++
			return errors.New("expected")
		}),
		Stop(func(_ Application) error {
			stopCount++
			return nil
		}),
		Stop(func(_ Application) error {
			stopCount++
			return nil
		}),
		Stop(func(_ Application) error {
			stopCount++
			return nil
		}),
	).(*dynamicModule)

	assert.Len(t, mod.init, 2)
	assert.Len(t, mod.start, 3)
	assert.Len(t, mod.stop, 5)
	assert.Len(t, mod.reload, 1)

	assert.Error(t, mod.Init(nil))
	assert.Error(t, mod.Start(nil))
	assert.Error(t, mod.Stop(nil))
	assert.Error(t, mod.Reload(nil))

	assert.Equal(t, 1, initCount)
	assert.Equal(t, 2, startCount)
	assert.Equal(t, 1, reloadCount)
	assert.Equal(t, 2, stopCount)
}
