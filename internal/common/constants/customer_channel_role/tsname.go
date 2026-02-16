package customer_channel_role

// TSName returns the TypeScript enum name for the CustomerChannelRole value.
func (x CustomerChannelRole) TSName() string {
	if str, ok := _CustomerChannelRoleMap[x]; ok {
		return str
	}
	return "???"
}
