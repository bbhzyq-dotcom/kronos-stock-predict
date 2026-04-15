package models

import "time"

type Stock struct {
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	Market    uint8     `json:"market"`
	Price     float64   `json:"price"`
	ChangePct float64   `json:"change_pct"`
	Volume    float64   `json:"volume"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Kline struct {
	Code      string    `json:"code"`
	Timestamp time.Time `json:"timestamp"`
	Open      float64   `json:"open"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Close     float64   `json:"close"`
	Volume    float64   `json:"volume"`
	Amount    float64   `json:"amount"`
}

type PredictionRecord struct {
	Code        string    `json:"code"`
	Lookback    int       `json:"lookback"`
	Direction   string    `json:"direction"`
	ChangePct   float64   `json:"change_pct"`
	Score       float64   `json:"score"`
	NextOpen    float64   `json:"next_open"`
	NextHigh    float64   `json:"next_high"`
	NextLow     float64   `json:"next_low"`
	NextClose   float64   `json:"next_close"`
	PredictedAt time.Time `json:"predicted_at"`
}

type StockPrediction struct {
	Stock       Stock               `json:"stock"`
	Predictions []PredictionDisplay `json:"predictions"`
}

type PredictionDisplay struct {
	Lookback    int       `json:"lookback"`
	Direction   string    `json:"direction"`
	ChangePct   float64   `json:"change_pct"`
	Score       float64   `json:"score"`
	NextOpen    float64   `json:"next_open"`
	NextHigh    float64   `json:"next_high"`
	NextLow     float64   `json:"next_low"`
	NextClose   float64   `json:"next_close"`
	PredictedAt time.Time `json:"predicted_at"`
}

type PredictionRequest struct {
	Code      string `json:"code" binding:"required"`
	Lookbacks []int  `json:"lookbacks"`
}

type PredictionResult struct {
	Code    string                    `json:"code"`
	Name    string                    `json:"name"`
	Predict map[string]PredictionItem `json:"predictions"`
}

type PredictionItem struct {
	NextKline KlineInfo `json:"next_kline"`
	Direction string    `json:"direction"`
	ChangePct float64   `json:"change_pct"`
	Score     float64   `json:"score"`
}

type KlineInfo struct {
	Open  float64 `json:"open"`
	High  float64 `json:"high"`
	Low   float64 `json:"low"`
	Close float64 `json:"close"`
}

type UpdateResponse struct {
	Code      string `json:"code"`
	Updated   int    `json:"updated"`
	Skipped   int    `json:"skipped"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

type SyncStatusResponse struct {
	LastSync    string `json:"last_sync"`
	Status      string `json:"status"`
	TotalStocks int    `json:"total_stocks"`
	Processed   int    `json:"processed"`
}
