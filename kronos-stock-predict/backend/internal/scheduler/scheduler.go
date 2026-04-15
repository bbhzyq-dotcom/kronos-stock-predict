package scheduler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"kronos-stock-predict/backend/internal/data"
	"kronos-stock-predict/backend/internal/gotdx"
	"kronos-stock-predict/backend/internal/models"
)

type Scheduler struct {
	db       *data.DB
	tdx      *gotdx.Client
	predURL  string
	running  bool
	mu       sync.Mutex
	stopChan chan struct{}
}

func NewScheduler(db *data.DB, tdx *gotdx.Client, predURL string) *Scheduler {
	return &Scheduler{
		db:       db,
		tdx:      tdx,
		predURL:  predURL,
		stopChan: make(chan struct{}),
	}
}

func (s *Scheduler) Start() {
	go s.run()
}

func (s *Scheduler) Stop() {
	close(s.stopChan)
}

func (s *Scheduler) run() {
	for {
		now := time.Now()
		next := s.nextRunTime(now)
		duration := next.Sub(now)

		log.Printf("[Scheduler] Next run scheduled at %s (in %s)", next.Format("2006-01-02 15:04:05"), duration)

		select {
		case <-time.After(duration):
			s.runSync()
		case <-s.stopChan:
			log.Printf("[Scheduler] Stopped")
			return
		}
	}
}

func (s *Scheduler) nextRunTime(now time.Time) time.Time {
	targetHour := 16
	targetMinute := 30

	next := time.Date(now.Year(), now.Month(), now.Day(), targetHour, targetMinute, 0, 0, now.Location())

	if next.Before(now) {
		next = next.AddDate(0, 0, 1)
	}

	return next
}

func (s *Scheduler) RunOnce() {
	go s.runSync()
}

func (s *Scheduler) runSync() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		log.Printf("[Scheduler] Sync already running, skipping")
		return
	}
	s.running = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()

	log.Printf("[Scheduler] Starting daily sync...")
	startTime := time.Now()

	allStocks, err := s.tdx.GetStockList()
	if err != nil {
		log.Printf("[Scheduler] Failed to get stock list: %v", err)
		s.db.UpdateSyncStatus("failed", 0, 0)
		return
	}

	s.db.UpdateSyncStatus("running", len(allStocks), 0)

	for i, stock := range allStocks {
		select {
		case <-s.stopChan:
			log.Printf("[Scheduler] Sync interrupted")
			return
		default:
		}

		if i%50 == 0 {
			s.db.UpdateSyncStatus("running", len(allStocks), i)
			time.Sleep(200 * time.Millisecond)
		}

		if err := s.db.UpsertStock(&stock); err != nil {
			continue
		}

		klines, err := s.tdx.GetKline(stock.Code, stock.Market, 150)
		if err != nil || len(klines) == 0 {
			continue
		}

		s.db.UpsertKlines(klines)

		stock.Price = klines[len(klines)-1].Close
		s.db.UpsertStock(&stock)

		s.calculateAndSavePredictions(stock.Code, klines)
	}

	s.db.UpdateSyncStatus("completed", len(allStocks), len(allStocks))

	log.Printf("[Scheduler] Sync completed in %s", time.Since(startTime))
}

func (s *Scheduler) calculateAndSavePredictions(code string, klines []models.Kline) {
	lookbacks := []int{120, 30, 10, 5}

	kronosReq := map[string]interface{}{
		"code":      code,
		"lookbacks": lookbacks,
		"klines":    klines,
	}

	reqBytes, _ := json.Marshal(kronosReq)
	resp, err := http.Post(s.predURL+"/predict", "application/json", bytes.NewBuffer(reqBytes))
	if err != nil {
		log.Printf("[Scheduler] Prediction request failed for %s: %v", code, err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return
	}

	var predResp struct {
		Code        string `json:"code"`
		Predictions map[string]struct {
			NextKline struct {
				Open  float64 `json:"open"`
				High  float64 `json:"high"`
				Low   float64 `json:"low"`
				Close float64 `json:"close"`
			} `json:"next_kline"`
			Direction string  `json:"direction"`
			ChangePct float64 `json:"change_pct"`
			Score     float64 `json:"score"`
		} `json:"predictions"`
	}

	if err := json.Unmarshal(body, &predResp); err != nil {
		return
	}

	for lb, pred := range predResp.Predictions {
		var lookback int
		fmt.Sscanf(lb, "%d", &lookback)

		record := &models.PredictionRecord{
			Code:      code,
			Lookback:  lookback,
			Direction: pred.Direction,
			ChangePct: pred.ChangePct,
			Score:     pred.Score,
			NextOpen:  pred.NextKline.Open,
			NextHigh:  pred.NextKline.High,
			NextLow:   pred.NextKline.Low,
			NextClose: pred.NextKline.Close,
		}
		s.db.UpsertPrediction(record)
	}
}
