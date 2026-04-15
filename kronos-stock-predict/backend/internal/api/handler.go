package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"kronos-stock-predict/backend/internal/data"
	"kronos-stock-predict/backend/internal/gotdx"
	"kronos-stock-predict/backend/internal/models"
	"kronos-stock-predict/backend/internal/scheduler"
)

type Handler struct {
	db        *data.DB
	tdx       *gotdx.Client
	scheduler *scheduler.Scheduler
}

func NewHandler(db *data.DB, tdx *gotdx.Client, predURL string) *Handler {
	h := &Handler{
		db:  db,
		tdx: tdx,
	}
	h.scheduler = scheduler.NewScheduler(db, tdx, predURL)
	return h
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	r.GET("/api/stocks", h.GetStocks)
	r.GET("/api/stock/:code", h.GetStock)
	r.GET("/api/kline/:code", h.GetKline)
	r.GET("/api/predictions", h.GetAllPredictions)
	r.GET("/api/prediction/:code", h.GetPrediction)
	r.GET("/api/sync/status", h.GetSyncStatus)
	r.POST("/api/sync/trigger", h.TriggerSync)
	r.GET("/api/health", h.Health)
}

func (h *Handler) GetStocks(c *gin.Context) {
	stocks, err := h.db.GetAllStocks()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(stocks) == 0 {
		stocks, err = h.refreshStockList()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, stocks)
}

func (h *Handler) GetStock(c *gin.Context) {
	code := c.Param("code")

	stock, err := h.db.GetStock(code)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "stock not found"})
		return
	}

	c.JSON(http.StatusOK, stock)
}

func (h *Handler) GetKline(c *gin.Context) {
	code := c.Param("code")
	limit := 150

	klines, err := h.db.GetKlines(code, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, klines)
}

func (h *Handler) GetAllPredictions(c *gin.Context) {
	predictions, err := h.db.GetAllPredictions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, predictions)
}

func (h *Handler) GetPrediction(c *gin.Context) {
	code := c.Param("code")

	stock, err := h.db.GetStock(code)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "stock not found"})
		return
	}

	predictions, err := h.db.GetPredictions(code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	result := models.StockPrediction{
		Stock:       *stock,
		Predictions: make([]models.PredictionDisplay, 0),
	}

	for _, p := range predictions {
		result.Predictions = append(result.Predictions, models.PredictionDisplay{
			Lookback:    p.Lookback,
			Direction:   p.Direction,
			ChangePct:   p.ChangePct,
			Score:       p.Score,
			NextOpen:    p.NextOpen,
			NextHigh:    p.NextHigh,
			NextLow:     p.NextLow,
			NextClose:   p.NextClose,
			PredictedAt: p.PredictedAt,
		})
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) GetSyncStatus(c *gin.Context) {
	lastSync, status, total, processed, err := h.db.GetSyncStatus()
	if err != nil {
		c.JSON(http.StatusOK, models.SyncStatusResponse{
			Status:      "idle",
			TotalStocks: 0,
			Processed:   0,
		})
		return
	}

	lastSyncStr := ""
	if !lastSync.IsZero() {
		lastSyncStr = lastSync.Format("2006-01-02 15:04:05")
	}

	c.JSON(http.StatusOK, models.SyncStatusResponse{
		LastSync:    lastSyncStr,
		Status:      status,
		TotalStocks: total,
		Processed:   processed,
	})
}

func (h *Handler) TriggerSync(c *gin.Context) {
	h.scheduler.RunOnce()
	c.JSON(http.StatusOK, gin.H{"message": "Sync triggered"})
}

func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) StartScheduler() {
	h.scheduler.Start()
}

func (h *Handler) StopScheduler() {
	h.scheduler.Stop()
}

func (h *Handler) refreshStockList() ([]models.Stock, error) {
	allStocks, err := h.tdx.GetStockList()
	if err != nil {
		return nil, err
	}

	for _, stock := range allStocks {
		h.db.UpsertStock(&stock)
	}

	return allStocks, nil
}
