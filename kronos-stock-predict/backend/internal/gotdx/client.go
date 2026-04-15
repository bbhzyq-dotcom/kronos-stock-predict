package gotdx

import (
	"fmt"
	"strings"
	"time"

	"github.com/bensema/gotdx"
	"kronos-stock-predict/backend/internal/models"
)

type Client struct {
	client *gotdx.Client
}

const (
	MarketSH = gotdx.MarketSH
	MarketSZ = gotdx.MarketSZ
	MarketBJ = gotdx.MarketBJ

	CategoryDaily = 4
)

var aSharePrefixes = []string{"60", "68", "00", "30"}

func NewClient() (*Client, error) {
	mainHosts := gotdx.MainHostAddresses()
	exHosts := gotdx.ExHostAddresses()

	client := gotdx.New(
		gotdx.WithTCPAddress(mainHosts[0]),
		gotdx.WithTCPAddressPool(mainHosts[1:]...),
		gotdx.WithExTCPAddress(exHosts[0]),
		gotdx.WithExTCPAddressPool(exHosts[1:]...),
		gotdx.WithAutoSelectFastest(true),
		gotdx.WithTimeoutSec(10),
	)

	return &Client{client: client}, nil
}

func (c *Client) Disconnect() {
	if c.client != nil {
		c.client.Disconnect()
	}
}

func isAShare(code string) bool {
	for _, prefix := range aSharePrefixes {
		if strings.HasPrefix(code, prefix) {
			return true
		}
	}
	return false
}

func (c *Client) GetStockList() ([]models.Stock, error) {
	var allStocks []models.Stock

	markets := []uint8{gotdx.MarketSH, gotdx.MarketSZ, gotdx.MarketBJ}

	for _, market := range markets {
		count, err := c.client.StockCount(market)
		if err != nil {
			continue
		}

		const batchSize = 800
		for start := uint32(0); start < uint32(count); start += batchSize {
			secList, err := c.client.StockList(market, start, batchSize)
			if err != nil {
				continue
			}

			for _, sec := range secList {
				if !isAShare(sec.Code) {
					continue
				}
				allStocks = append(allStocks, models.Stock{
					Code:   sec.Code,
					Name:   sec.Name,
					Market: market,
					Price:  0,
				})
			}
		}
	}

	return allStocks, nil
}

func (c *Client) GetStockQuotes(codes []string) ([]models.Stock, error) {
	if len(codes) == 0 {
		return nil, nil
	}

	markets := make([]uint8, len(codes))
	stockCodes := make([]string, len(codes))

	for i, code := range codes {
		if len(code) >= 1 && (code[0] == '6' || code[0] == '9') {
			markets[i] = MarketSH
		} else if len(code) >= 1 && (code[0] == '8' || code[0] == '4') {
			markets[i] = MarketBJ
		} else {
			markets[i] = MarketSZ
		}
		stockCodes[i] = code
	}

	quotes, err := c.client.StockQuotesDetail(markets, stockCodes)
	if err != nil {
		return nil, err
	}

	stocks := make([]models.Stock, 0, len(quotes))
	for _, q := range quotes {
		market := MarketSZ
		if q.Market == 1 {
			market = MarketSH
		} else if q.Market == 2 {
			market = MarketBJ
		}

		stocks = append(stocks, models.Stock{
			Code:      q.Code,
			Name:      "",
			Market:    market,
			Price:     q.Price,
			ChangePct: q.Rate,
			Volume:    float64(q.Vol),
		})
	}

	return stocks, nil
}

func (c *Client) GetKline(code string, market uint8, count int) ([]models.Kline, error) {
	securityBars, err := c.client.StockKLine(CategoryDaily, market, code, 0, uint16(count), 1, 0)
	if err != nil {
		return nil, err
	}

	if len(securityBars) == 0 {
		return nil, err
	}

	klines := make([]models.Kline, 0, len(securityBars))
	for _, bar := range securityBars {
		ts, err := parseTimestamp(bar.DateTime)
		if err != nil {
			continue
		}

		klines = append(klines, models.Kline{
			Code:      code,
			Timestamp: ts,
			Open:      bar.Open,
			High:      bar.High,
			Low:       bar.Low,
			Close:     bar.Close,
			Volume:    bar.Vol,
			Amount:    bar.Amount,
		})
	}

	return klines, nil
}

func parseTimestamp(s string) (time.Time, error) {
	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02",
		"2006/01/02 15:04:05",
		"2006/01/02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("failed to parse timestamp: %s", s)
}
