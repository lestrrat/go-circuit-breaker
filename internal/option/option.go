package option

type Value struct {
	name  string
	value interface{}
}

func NewValue(name string, value interface{}) *Value {
	return &Value{
		name: name,
		value: value,
	}
}

func (v *Value) Name() string {
	return v.name
}

func (v *Value) Get() interface{} {
	return v.value
}


