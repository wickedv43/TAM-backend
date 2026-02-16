package deal_type

// TSName returns the TypeScript enum name for the DealType value.
func (x DealType) TSName() string {
	if str, ok := _DealTypeMap[x]; ok {
		return str
	}
	return "???"
}
