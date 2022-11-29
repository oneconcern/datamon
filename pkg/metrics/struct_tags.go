package metrics

import (
	"fmt"
	"path"
	"reflect"
)

type metricAdder func(interface{}, string, string, map[string]string) interface{}

func equalType(a, b interface{}) bool {
	hadThis := reflect.TypeOf(a)
	nowThis := reflect.TypeOf(b)
	return hadThis == nowThis
}

func scanStruct(parent string, adder metricAdder, m interface{}) {
	rv := reflect.ValueOf(m)
	if rv.Kind() != reflect.Ptr || rv.IsNil() || rv.Elem().Type().Kind() != reflect.Struct {
		panic(fmt.Sprintf("scanStruct requires a pointer to a struct, got: %T", m))
	}
	scanTags(parent, adder, m)
}

func scanTags(parent string, adder metricAdder, m interface{}) {
	container := reflect.ValueOf(m)
	if shouldSkip(container) || !isStruct(container) {
		return
	}

	receiver, derefType := pointerTo(container) // always a pointer
	pointedStruct := reflect.Indirect(receiver) // always a struct
	structChanged := false

	for i := 0; i < derefType.NumField(); i++ {
		field := derefType.Field(i)
		pointedField := pointedStruct.Field(i)

		if !pointedField.CanInterface() {
			continue
		}
		var child interface{}
		if pointedField.Type().Kind() == reflect.Ptr {
			child = pointedField.Interface()
		} else {
			child = pointedField.Addr().Interface()
		}

		tags := fieldTags(field)
		metric := tags["metric"]
		group := tags["group"]

		if metric == "" {
			scanTags(path.Join(parent, group), adder, child)
			continue
		}

		if skipMetric(pointedField) {
			continue
		}

		allocated := adder(child, metric, path.Join(parent, group), tags)
		if allocated != nil {
			pointedField.Set(reflect.ValueOf(allocated))
			structChanged = true
		}
	}

	if !structChanged {
		return
	}

	if container.CanSet() {
		container.Set(receiver)
	} else if container.CanAddr() && container.Addr().CanSet() {
		container.Addr().Set(receiver)
	}
}

func pointerTo(container reflect.Value) (reflect.Value, reflect.Type) {
	var receiver reflect.Value
	deref := derefType(container)
	if container.Type().Kind() == reflect.Ptr {
		if container.IsNil() {
			receiver = reflect.New(deref)
		} else {
			receiver = container
		}
	} else {
		receiver = reflect.New(deref)
	}
	return receiver, deref
}

func skipMetric(field reflect.Value) bool {
	return field.Type().Kind() != reflect.Ptr || !field.CanSet()
}

// fieldTags decodes field tags that decorate the struct.
// Supported tags are:
//   - metric: the metric name
//   - group: builds an additional path to the metric (e.g.  root/path/mymetrics/{metric})
//   - description: adds this description to the metric and the associated views
//   - extraviews:[aggregator, ...]: builds additional views with alternate aggregators
func fieldTags(field reflect.StructField) map[string]string {
	tags := make(map[string]string, 5)
	if metric, ok := field.Tag.Lookup("metric"); ok {
		tags["metric"] = metric
	}
	if unit, ok := field.Tag.Lookup("unit"); ok {
		tags["unit"] = unit
	}
	if group, ok := field.Tag.Lookup("group"); ok {
		tags["group"] = group
	}
	if description, ok := field.Tag.Lookup("description"); ok {
		tags["description"] = description
	}
	if views, ok := field.Tag.Lookup("extraviews"); ok {
		tags["views"] = views
	}
	if groupings, ok := field.Tag.Lookup("tags"); ok {
		tags["groupings"] = groupings
	}
	return tags
}

func isStruct(v reflect.Value) bool {
	return (v.Type().Kind() == reflect.Ptr && v.Type().Elem().Kind() == reflect.Struct) || (v.Type().Kind() == reflect.Struct)
}

func derefType(v reflect.Value) reflect.Type {
	if v.Type().Kind() == reflect.Ptr {
		return v.Type().Elem()
	}
	return v.Type()
}

func shouldSkip(v reflect.Value) bool {
	return !v.IsValid() || v.Type().Kind() == reflect.Map || v.Type().Kind() == reflect.Map
}
