package channel_status

// TSName returns the TypeScript enum name for the ChannelStatus value.
func (x ChannelStatus) TSName() string {
	if str, ok := _ChannelStatusMap[x]; ok {
		return str
	}
	return "???"
}
