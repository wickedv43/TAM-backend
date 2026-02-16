package transaction_status

// TSName returns the TypeScript enum name for the TransactionStatus value.
func (x TransactionStatus) TSName() string {
	if str, ok := _TransactionStatusMap[x]; ok {
		return str
	}
	return "???"
}
