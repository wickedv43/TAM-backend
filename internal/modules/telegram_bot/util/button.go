package util

type UniqueCallbacker interface {
	Unique() string
	CallbackUnique() string
}

type CallbackUnique string

func (u CallbackUnique) Unique() string {
	return string(u)
}

func (u CallbackUnique) CallbackUnique() string {
	return "\f" + string(u)
}
