package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"

	"kronos-stock-predict/backend/internal/api"
	"kronos-stock-predict/backend/internal/data"
	"kronos-stock-predict/backend/internal/gotdx"
)

type Config struct {
	Server     ServerConfig     `yaml:"server"`
	Database   DatabaseConfig   `yaml:"database"`
	Prediction PredictionConfig `yaml:"prediction"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type DatabaseConfig struct {
	Path string `yaml:"path"`
}

type PredictionConfig struct {
	URL string `yaml:"url"`
}

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	cfg, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	db, err := data.NewDB(cfg.Database.Path)
	if err != nil {
		log.Fatalf("init db: %v", err)
	}
	defer db.Close()

	tdx, err := gotdx.NewClient()
	if err != nil {
		log.Fatalf("init tdx client: %v", err)
	}
	defer tdx.Disconnect()

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	r.Static("/static", "./static")
	r.GET("/", func(c *gin.Context) {
		c.File("./static/index.html")
	})

	h := api.NewHandler(db, tdx, cfg.Prediction.URL)
	h.RegisterRoutes(r)

	h.StartScheduler()
	defer h.StopScheduler()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Printf("Shutting down scheduler...")
		h.StopScheduler()
		os.Exit(0)
	}()

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("server starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	if cfg.Database.Path != "" && !filepath.IsAbs(cfg.Database.Path) {
		execPath, _ := os.Executable()
		baseDir := filepath.Dir(execPath)
		cfg.Database.Path = filepath.Join(baseDir, cfg.Database.Path)
	}

	return &cfg, nil
}
