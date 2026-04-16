package data

import (
	"database/sql"
	"strings"
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
	CREATE INDEX IF NOT EXISTS idx_predictions_code_date ON predictions(code, predicted_date);
	`
	_, err := db.conn.Exec(query)
	return err
}

func (db *PredictionDB) Close() error {
	return db.conn.Close()
}

func parsePredictedAt(s string) time.Time {
	s = strings.ReplaceAll(s, " ", "T")
	if strings.HasSuffix(s, "Z") {
		s = s[:len(s)-1] + "+00:00"
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
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
		p.PredictedAt = parsePredictedAt(predictedAtStr)
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
		p.PredictedAt = parsePredictedAt(predictedAtStr)
		preds = append(preds, p)
	}
	return preds, rows.Err()
}

type PredictionWithName struct {
	Code        string
	Name        string
	Lookback    int
	Direction   string
	ChangePct   float64
	Score       float64
	NextOpen    float64
	NextHigh    float64
	NextLow     float64
	NextClose   float64
	PredictedAt string
}

func (db *PredictionDB) GetAllLatestPredictions() ([]PredictionWithName, error) {
	query := `
	SELECT p.code, p.lookback, p.direction, p.change_pct, p.score, p.next_open, p.next_high, p.next_low, p.next_close, p.predicted_at
	FROM predictions p
	WHERE (p.code, p.predicted_date) IN (
		SELECT code, MAX(predicted_date)
		FROM predictions
		GROUP BY code
	)
	ORDER BY p.code, p.lookback
	`
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []PredictionWithName
	for rows.Next() {
		var p PredictionWithName
		if err := rows.Scan(&p.Code, &p.Lookback, &p.Direction, &p.ChangePct, &p.Score, &p.NextOpen, &p.NextHigh, &p.NextLow, &p.NextClose, &p.PredictedAt); err != nil {
			continue
		}
		results = append(results, p)
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
