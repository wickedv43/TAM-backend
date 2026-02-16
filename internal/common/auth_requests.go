package common

// --- Auth ---
type AuthRequest struct {
	InitData string `json:"init_data"`
	RefID    string `json:"ref_id"`
}

type AuthResponse struct {
	Token string `json:"token"`
}
