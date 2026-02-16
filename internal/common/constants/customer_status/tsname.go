package customer_status

// TSName returns the TypeScript enum name for the CustomerStatus value.
func (x CustomerStatus) TSName() string {
	if str, ok := _CustomerStatusMap[x]; ok {
		return str
	}
	return "???"
}
