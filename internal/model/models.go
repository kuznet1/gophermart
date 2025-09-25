package model

import "time"

type UserCredentials struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type Order struct {
	Order      int       `json:"number,string"`
	Status     string    `json:"status"`
	Accrual    float64   `json:"accrual,omitempty"`
	UploadedAt time.Time `json:"uploaded_at"`
}

type Withdraw struct {
	Order int     `json:"order,string"`
	Sum   float64 `json:"sum"`
}

type Withdrawal struct {
	Order       int       `json:"order,string"`
	Sum         float64   `json:"sum"`
	ProcessedAt time.Time `json:"processed_at"`
}

type Balance struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

type AccrualResp struct {
	Order   int     `json:"order,string"`
	Status  string  `json:"status"`
	Accrual float64 `json:"accrual,omitempty"`
}
