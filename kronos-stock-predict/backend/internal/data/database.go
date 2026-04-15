package data

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"kronos-stock-predict/backend/internal/models"
)

type DB struct {
	conn *sql.DB
}

func NewDB(dbPath string) (*DB, error) {
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return db, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS stocks (
		code TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		market INTEGER NOT NULL,
		price REAL DEFAULT 0,
		change_pct REAL DEFAULT 0,
		volume REAL DEFAULT 0,
		updated_at TEXT DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS klines (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		code TEXT NOT NULL,
		timestamp TEXT NOT NULL,
		open REAL NOT NULL,
		high REAL NOT NULL,
		low REAL NOT NULL,
		close REAL NOT NULL,
		volume REAL NOT NULL,
		amount REAL DEFAULT 0,
		UNIQUE(code, timestamp)
	);

	CREATE TABLE IF NOT EXISTS predictions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		code TEXT NOT NULL,
		lookback INTEGER NOT NULL,
		direction TEXT NOT NULL,
		change_pct REAL NOT NULL,
		score REAL NOT NULL,
		next_open REAL NOT NULL,
		next_high REAL NOT NULL,
		next_low REAL NOT NULL,
		next_close REAL NOT NULL,
		predicted_at TEXT DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(code, lookback)
	);

	CREATE TABLE IF NOT EXISTS sync_status (
		id INTEGER PRIMARY KEY,
		last_sync TEXT,
		status TEXT DEFAULT 'idle',
		total_stocks INTEGER DEFAULT 0,
		processed_stocks INTEGER DEFAULT 0
	);

	CREATE INDEX IF NOT EXISTS idx_klines_code_time ON klines(code, timestamp);
	CREATE INDEX IF NOT EXISTS idx_predictions_code ON predictions(code);
	`

	_, err := db.conn.Exec(schema)
	return err
}

func (db *DB) UpsertStock(stock *models.Stock) error {
	query := `
	INSERT INTO stocks (code, name, market, price, change_pct, volume, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(code) DO UPDATE SET
		name = excluded.name,
		market = excluded.market,
		price = excluded.price,
		change_pct = excluded.change_pct,
		volume = excluded.volume,
		updated_at = excluded.updated_at
	`
	_, err := db.conn.Exec(query, stock.Code, stock.Name, stock.Market, stock.Price, stock.ChangePct, stock.Volume, time.Now())
	return err
}

func (db *DB) GetAllStocks() ([]models.Stock, error) {
	query := `SELECT code, name, market, price, change_pct, volume, updated_at FROM stocks ORDER BY code`
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stocks []models.Stock
	for rows.Next() {
		var s models.Stock
		var updatedAtStr string
		if err := rows.Scan(&s.Code, &s.Name, &s.Market, &s.Price, &s.ChangePct, &s.Volume, &updatedAtStr); err != nil {
			return nil, err
		}
		s.UpdatedAt, _ = time.ParseInLocation(time.RFC3339, updatedAtStr, time.UTC)
		stocks = append(stocks, s)
	}
	return stocks, rows.Err()
}

func (db *DB) GetStock(code string) (*models.Stock, error) {
	query := `SELECT code, name, market, price, change_pct, volume, updated_at FROM stocks WHERE code = ?`
	var s models.Stock
	var updatedAtStr string
	err := db.conn.QueryRow(query, code).Scan(&s.Code, &s.Name, &s.Market, &s.Price, &s.ChangePct, &s.Volume, &updatedAtStr)
	if err != nil {
		return nil, err
	}
	s.UpdatedAt, _ = time.ParseInLocation(time.RFC3339, updatedAtStr, time.UTC)
	return &s, nil
}

func (db *DB) GetLatestKlineDate(code string) (time.Time, error) {
	query := `SELECT timestamp FROM klines WHERE code = ? ORDER BY timestamp DESC LIMIT 1`
	var tsStr sql.NullString
	err := db.conn.QueryRow(query, code).Scan(&tsStr)
	if err != nil {
		return time.Time{}, err
	}
	if !tsStr.Valid {
		return time.Time{}, nil
	}
	return time.ParseInLocation(time.RFC3339, tsStr.String, time.UTC)
}

func (db *DB) GetKlineDateRange(code string) (start, end time.Time, err error) {
	queryStart := `SELECT timestamp FROM klines WHERE code = ? ORDER BY timestamp ASC LIMIT 1`
	queryEnd := `SELECT timestamp FROM klines WHERE code = ? ORDER BY timestamp DESC LIMIT 1`

	var startStr, endStr sql.NullString
	err = db.conn.QueryRow(queryStart, code).Scan(&startStr)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	db.conn.QueryRow(queryEnd, code).Scan(&endStr)

	if startStr.Valid {
		start, _ = time.ParseInLocation(time.RFC3339, startStr.String, time.UTC)
	}
	if endStr.Valid {
		end, _ = time.ParseInLocation(time.RFC3339, endStr.String, time.UTC)
	}
	return start, end, nil
}

func (db *DB) UpsertKline(kline *models.Kline) error {
	query := `
	INSERT INTO klines (code, timestamp, open, high, low, close, volume, amount)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(code, timestamp) DO UPDATE SET
		open = excluded.open,
		high = excluded.high,
		low = excluded.low,
		close = excluded.close,
		volume = excluded.volume,
		amount = excluded.amount
	`
	tsStr := kline.Timestamp.Format("2006-01-02 15:04:05")
	_, err := db.conn.Exec(query, kline.Code, tsStr, kline.Open, kline.High, kline.Low, kline.Close, kline.Volume, kline.Amount)
	return err
}

func (db *DB) UpsertKlines(klines []models.Kline) (updated, skipped int, err error) {
	tx, err := db.conn.Begin()
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO klines (code, timestamp, open, high, low, close, volume, amount)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(code, timestamp) DO UPDATE SET
			open = excluded.open,
			high = excluded.high,
			low = excluded.low,
			close = excluded.close,
			volume = excluded.volume,
			amount = excluded.amount
	`)
	if err != nil {
		return 0, 0, err
	}
	defer stmt.Close()

	for _, k := range klines {
		tsStr := k.Timestamp.Format(time.RFC3339)
		result, err := stmt.Exec(k.Code, tsStr, k.Open, k.High, k.Low, k.Close, k.Volume, k.Amount)
		if err != nil {
			return updated, skipped, err
		}
		rows, _ := result.RowsAffected()
		if rows > 0 {
			updated++
		} else {
			skipped++
		}
	}

	return updated, skipped, tx.Commit()
}

func (db *DB) GetKlines(code string, limit int) ([]models.Kline, error) {
	query := `
	SELECT code, timestamp, open, high, low, close, volume, amount 
	FROM klines 
	WHERE code = ? 
	ORDER BY timestamp DESC 
	LIMIT ?
	`
	rows, err := db.conn.Query(query, code, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var klines []models.Kline
	for rows.Next() {
		var k models.Kline
		var tsStr string
		if err := rows.Scan(&k.Code, &tsStr, &k.Open, &k.High, &k.Low, &k.Close, &k.Volume, &k.Amount); err != nil {
			return nil, err
		}
		k.Timestamp, _ = time.Parse(time.RFC3339, tsStr)
		klines = append(klines, k)
	}

	for i, j := 0, len(klines)-1; i < j; i, j = i+1, j-1 {
		klines[i], klines[j] = klines[j], klines[i]
	}

	return klines, rows.Err()
}

func (db *DB) GetKlineCount(code string) (int, error) {
	query := `SELECT COUNT(*) FROM klines WHERE code = ?`
	var count int
	err := db.conn.QueryRow(query, code).Scan(&count)
	return count, err
}

func (db *DB) UpsertPrediction(pred *models.PredictionRecord) error {
	query := `
	INSERT INTO predictions (code, lookback, direction, change_pct, score, next_open, next_high, next_low, next_close, predicted_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(code, lookback) DO UPDATE SET
		direction = excluded.direction,
		change_pct = excluded.change_pct,
		score = excluded.score,
		next_open = excluded.next_open,
		next_high = excluded.next_high,
		next_low = excluded.next_low,
		next_close = excluded.next_close,
		predicted_at = excluded.predicted_at
	`
	_, err := db.conn.Exec(query, pred.Code, pred.Lookback, pred.Direction, pred.ChangePct, pred.Score, pred.NextOpen, pred.NextHigh, pred.NextLow, pred.NextClose, time.Now())
	return err
}

func (db *DB) GetPredictions(code string) ([]models.PredictionRecord, error) {
	query := `
	SELECT code, lookback, direction, change_pct, score, next_open, next_high, next_low, next_close, predicted_at
	FROM predictions
	WHERE code = ?
	ORDER BY lookback
	`
	rows, err := db.conn.Query(query, code)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var preds []models.PredictionRecord
	for rows.Next() {
		var p models.PredictionRecord
		var predictedAtStr string
		if err := rows.Scan(&p.Code, &p.Lookback, &p.Direction, &p.ChangePct, &p.Score, &p.NextOpen, &p.NextHigh, &p.NextLow, &p.NextClose, &predictedAtStr); err != nil {
			return nil, err
		}
		p.PredictedAt, _ = time.Parse("2006-01-02T15:04:05Z07:00", predictedAtStr)
		preds = append(preds, p)
	}
	return preds, rows.Err()
}

func (db *DB) GetAllPredictions() ([]models.StockPrediction, error) {
	query := `
	SELECT s.code, s.name, s.market, s.price, s.change_pct,
		   p.lookback, p.direction, p.change_pct, p.score, p.next_open, p.next_high, p.next_low, p.next_close, p.predicted_at
	FROM stocks s
	LEFT JOIN predictions p ON s.code = p.code
	WHERE p.id IS NOT NULL
	ORDER BY s.code, p.lookback
	`
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.StockPrediction
	currentStock := &models.StockPrediction{}
	var lastCode string

	for rows.Next() {
		var code, name string
		var market uint8
		var price, changePct float64
		var lookback int
		var direction string
		var predChangePct, score float64
		var nextOpen, nextHigh, nextLow, nextClose float64
		var predictedAtStr string

		err := rows.Scan(&code, &name, &market, &price, &changePct, &lookback, &direction, &predChangePct, &score, &nextOpen, &nextHigh, &nextLow, &nextClose, &predictedAtStr)
		if err != nil {
			continue
		}

		predictedAt, _ := time.Parse("2006-01-02T15:04:05Z07:00", predictedAtStr)

		if code != lastCode {
			if currentStock.Stock.Code != "" {
				results = append(results, *currentStock)
			}
			currentStock = &models.StockPrediction{
				Stock: models.Stock{
					Code:      code,
					Name:      name,
					Market:    market,
					Price:     price,
					ChangePct: changePct,
				},
				Predictions: make([]models.PredictionDisplay, 0),
			}
			lastCode = code
		}

		currentStock.Predictions = append(currentStock.Predictions, models.PredictionDisplay{
			Lookback:    lookback,
			Direction:   direction,
			ChangePct:   predChangePct,
			Score:       score,
			NextOpen:    nextOpen,
			NextHigh:    nextHigh,
			NextLow:     nextLow,
			NextClose:   nextClose,
			PredictedAt: predictedAt,
		})
	}

	if currentStock.Stock.Code != "" {
		results = append(results, *currentStock)
	}

	return results, rows.Err()
}

func (db *DB) UpdateSyncStatus(status string, total, processed int) error {
	query := `INSERT OR REPLACE INTO sync_status (id, last_sync, status, total_stocks, processed_stocks) VALUES (1, ?, ?, ?, ?)`
	_, err := db.conn.Exec(query, time.Now(), status, total, processed)
	return err
}

func (db *DB) GetSyncStatus() (lastSync time.Time, status string, total, processed int, err error) {
	query := `SELECT last_sync, status, total_stocks, processed_stocks FROM sync_status WHERE id = 1`
	var lastSyncTs sql.NullTime
	err = db.conn.QueryRow(query).Scan(&lastSyncTs, &status, &total, &processed)
	if err != nil {
		return time.Time{}, "idle", 0, 0, err
	}
	if lastSyncTs.Valid {
		lastSync = lastSyncTs.Time
	}
	return lastSync, status, total, processed, nil
}

func (db *DB) ClearPredictions() error {
	_, err := db.conn.Exec("DELETE FROM predictions")
	return err
}
