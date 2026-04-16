package data

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"kronos-stock-predict/backend/internal/models"
)

type PredictionDB struct {
	conn *sql.DB
}

func NewPredictionDB(dbPath string) (*PredictionDB, error) {
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	db := &PredictionDB{conn: conn}
	if err := db.init(); err != nil {
		return nil, err
	}

	return db, nil
}

func (db *PredictionDB) init() error {
	query := `
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
		predicted_date TEXT NOT NULL,
		predicted_at TEXT NOT NULL,
		UNIQUE(code, lookback, predicted_date)
	);
	CREATE INDEX IF NOT EXISTS idx_predictions_code ON predictions(code);
	CREATE INDEX IF NOT EXISTS idx_predictions_date ON predictions(predicted_date);
	`
	_, err := db.conn.Exec(query)
	return err
}

func (db *PredictionDB) Close() error {
	return db.conn.Close()
}

func (db *PredictionDB) UpsertPrediction(record *models.PredictionRecord) error {
	predictedDate := time.Now().Format("2006-01-02")
	predictedAt := time.Now().Format(time.RFC3339)

	query := `
	INSERT INTO predictions (code, lookback, direction, change_pct, score, next_open, next_high, next_low, next_close, predicted_date, predicted_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(code, lookback, predicted_date) DO UPDATE SET
		direction = excluded.direction,
		change_pct = excluded.change_pct,
		score = excluded.score,
		next_open = excluded.next_open,
		next_high = excluded.next_high,
		next_low = excluded.next_low,
		next_close = excluded.next_close,
		predicted_at = excluded.predicted_at
	`
	_, err := db.conn.Exec(query, record.Code, record.Lookback, record.Direction, record.ChangePct, record.Score, record.NextOpen, record.NextHigh, record.NextLow, record.NextClose, predictedDate, predictedAt)
	return err
}

func (db *PredictionDB) GetPredictionsByDate(code string, date string) ([]models.PredictionRecord, error) {
	query := `
	SELECT code, lookback, direction, change_pct, score, next_open, next_high, next_low, next_close, predicted_at
	FROM predictions
	WHERE code = ? AND predicted_date = ?
	ORDER BY lookback
	`
	rows, err := db.conn.Query(query, code, date)
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
		p.PredictedAt, _ = time.Parse(time.RFC3339, predictedAtStr)
		preds = append(preds, p)
	}
	return preds, rows.Err()
}

func (db *PredictionDB) GetLatestPredictions(code string) ([]models.PredictionRecord, error) {
	query := `
	SELECT code, lookback, direction, change_pct, score, next_open, next_high, next_low, next_close, predicted_at
	FROM predictions
	WHERE code = ? AND predicted_date = (SELECT MAX(predicted_date) FROM predictions WHERE code = ?)
	ORDER BY lookback
	`
	rows, err := db.conn.Query(query, code, code)
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
		p.PredictedAt, _ = time.Parse(time.RFC3339, predictedAtStr)
		preds = append(preds, p)
	}
	return preds, rows.Err()
}

func (db *PredictionDB) GetAllLatestPredictions() ([]models.StockPrediction, error) {
	query := `
	SELECT p.code, p.lookback, p.direction, p.change_pct, p.score, p.next_open, p.next_high, p.next_low, p.next_close, p.predicted_at, s.name
	FROM predictions p
	JOIN (
		SELECT code, MAX(predicted_date) as max_date
		FROM predictions
		GROUP BY code
	) latest ON p.code = latest.code AND p.predicted_date = latest.max_date
	LEFT JOIN (
		SELECT code, name FROM stocks
	) s ON p.code = s.code
	ORDER BY p.code, p.lookback
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
		var lookback int
		var direction string
		var predChangePct, score float64
		var nextOpen, nextHigh, nextLow, nextClose float64
		var predictedAtStr string

		if err := rows.Scan(&code, &lookback, &direction, &predChangePct, &score, &nextOpen, &nextHigh, &nextLow, &nextClose, &predictedAtStr, &name); err != nil {
			continue
		}

		if code != lastCode {
			if currentStock.Stock.Code != "" {
				results = append(results, *currentStock)
			}
			currentStock = &models.StockPrediction{
				Stock: models.Stock{
					Code: code,
					Name: name,
				},
				Predictions: make([]models.PredictionDisplay, 0),
			}
			lastCode = code
		}

		predictedAt, _ := time.Parse(time.RFC3339, predictedAtStr)
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

func (db *PredictionDB) GetAvailableDates() ([]string, error) {
	query := `SELECT DISTINCT predicted_date FROM predictions ORDER BY predicted_date DESC`
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dates []string
	for rows.Next() {
		var date string
		if err := rows.Scan(&date); err != nil {
			continue
		}
		dates = append(dates, date)
	}
	return dates, rows.Err()
}

func (db *PredictionDB) ClearOldPredictions(keepDays int) error {
	cutoffDate := time.Now().AddDate(0, 0, -keepDays).Format("2006-01-02")
	query := `DELETE FROM predictions WHERE predicted_date < ?`
	_, err := db.conn.Exec(query, cutoffDate)
	return err
}

func (db *PredictionDB) GetPredictionStats() (int, int, error) {
	var total, dates int
	query := `SELECT COUNT(*) FROM predictions`
	if err := db.conn.QueryRow(query).Scan(&total); err != nil {
		return 0, 0, err
	}
	query = `SELECT COUNT(DISTINCT predicted_date) FROM predictions`
	if err := db.conn.QueryRow(query).Scan(&dates); err != nil {
		return 0, 0, err
	}
	return total, dates, nil
}
