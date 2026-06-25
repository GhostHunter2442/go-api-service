package dto

// LoginRequest body ของ POST /auth/login
type LoginRequest struct {
	PhoneNumber string `json:"phone_number" binding:"required"`
	Password    string `json:"password" binding:"required"`
}

// TokenResponse ส่ง access token กลับใน body (refresh อยู่ใน httpOnly cookie)
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"` // "Bearer"
	ExpiresIn   int    `json:"expires_in"` // วินาที
}
