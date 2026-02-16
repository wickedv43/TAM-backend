package deal_status

// TSName returns the TypeScript enum name for the DealStatus value.
func (x DealStatus) TSName() string {
	if str, ok := _DealStatusMap[x]; ok {
		return str
	}
	return "???"
}
