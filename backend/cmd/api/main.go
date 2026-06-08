package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Takapon2512/recipe-recommend/backend/internal/anthropic"
	"github.com/Takapon2512/recipe-recommend/backend/internal/config"
	"github.com/Takapon2512/recipe-recommend/backend/internal/handler"
	"github.com/Takapon2512/recipe-recommend/backend/internal/repository"
	"github.com/Takapon2512/recipe-recommend/backend/internal/service"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	// ロガー初期化
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// 設定読み込み
	cfg, err := config.Load()
	if err != nil {
		slog.Error("設定読み込み失敗", "error", err)
		os.Exit(1)
	}

	// DB接続
	db, err := gorm.Open(mysql.Open(cfg.DBDSN), &gorm.Config{})
	if err != nil {
		slog.Error("接続失敗", "error", err)
		os.Exit(1)
	}

	sqlDB, err := db.DB()
	if err != nil {
		slog.Error("DB取得失敗", "error", err)
		os.Exit(1)
	}

	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(time.Hour)
	defer func() {
		if err := sqlDB.Close(); err != nil {
			slog.Error("DB切断エラー", "error", err)
		}
	}()

	// レコメンドサービス構築
	llmClient := anthropic.NewClient(anthropic.ClientConfig{
		APIKey: cfg.AnthropicAPIKey,
		Model:  cfg.AnthropicModel,
	})
	recRepo := repository.NewRecommendationRepository(db)
	invRepo := repository.NewInventoryRepository(db)
	recService := service.NewRecommendationService(recRepo, invRepo, llmClient)

	// ルーター構築
	r := handler.NewRouter(cfg, db, recService)

	// レコメンドワーカー起動（pending ジョブを定期処理）
	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := recService.ProcessJob(workerCtx); err != nil {
					slog.Error("recommendation worker error", "error", err)
				}
			case <-workerCtx.Done():
				return
			}
		}
	}()

	// サーバ起動
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second, // LLM応答を考慮
		IdleTimeout:  120 * time.Second,
	}

	// グレースフルシャットダウン
	go func() {
		slog.Info("サーバ起動", "port", cfg.Port, "env", cfg.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("サーバエラー", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("シャットダウン開始...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("シャットダウンエラー", "error", err)
	}
	slog.Info("シャットダウン完了")
}
