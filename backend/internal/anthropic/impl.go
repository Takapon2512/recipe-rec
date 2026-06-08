package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Takapon2512/recipe-recommend/backend/internal/model"
)

// ----------------------------------------------------------------
// 定数
// ----------------------------------------------------------------

const (
	apiEndpoint  = "https://api.anthropic.com/v1/messages"
	apiVersion   = "2023-06-01"
	defaultModel = "claude-sonnet-4-6"
	maxTokens    = 4096
)

// ----------------------------------------------------------------
// コンストラクタ
// ----------------------------------------------------------------

// ClientConfig は Claude API クライアントの設定。
type ClientConfig struct {
	APIKey string
	// Model は使用するモデルID。空の場合は defaultModel を使う。
	Model string
	// HTTPClient は省略時に http.DefaultClient を使う。
	HTTPClient *http.Client
}

type clientImpl struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

// ----------------------------------------------------------------
// Anthropic API の型定義
// ----------------------------------------------------------------

type apiRequest struct {
	Model     string       `json:"model"`
	MaxTokens int          `json:"max_tokens"`
	System    string       `json:"system"`
	Messages  []apiMessage `json:"messages"`
}

type apiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type apiResponse struct {
	Content []apiContent `json:"content"`
}

type apiContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// NewClient は Client の具体実装を返す。
// 環境変数 ANTHROPIC_API_KEY / ANTHROPIC_MODEL を読み込んだ ClientConfig を渡す。
func NewClient(cfg ClientConfig) Client {
	m := cfg.Model
	if m == "" {
		m = defaultModel
	}
	hc := cfg.HTTPClient
	if hc == nil {
		hc = &http.Client{}
	}
	return &clientImpl{
		apiKey:     cfg.APIKey,
		model:      m,
		httpClient: hc,
	}
}

// ----------------------------------------------------------------
// Recommend の実装
// ----------------------------------------------------------------

func (c *clientImpl) Recommend(ctx context.Context, req *RecommendRequest) (*model.RecommendationResultPayload, error) {
	// --- 1. プロンプト組み立て ---
	systemPrompt := buildSystemPrompt()
	userPrompt := buildUserPrompt(req)

	// --- 2. API リクエスト構築 ---
	body, err := json.Marshal(apiRequest{
		Model:     c.model,
		MaxTokens: maxTokens,
		System:    systemPrompt,
		Messages: []apiMessage{
			{Role: "user", Content: userPrompt},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("anthropic: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiEndpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("anthropic: new request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", apiVersion)

	// --- 3. HTTP 呼び出し ---
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("anthropic: http do: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			_ = err
		}
	}()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("anthropic: read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("anthropic: unexpected status %d: %s", resp.StatusCode, string(respBody))
	}

	// --- 4. レスポンスのデシリアライズ ---
	var apiResp apiResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("anthropic: unmarshal response: %w", err)
	}

	text := extractText(apiResp)
	if text == "" {
		return nil, fmt.Errorf("anthropic: empty response content")
	}

	// --- 5. JSON パース ---
	result, err := parseResult(text)
	if err != nil {
		return nil, fmt.Errorf("anthropic: parse result: %w", err)
	}
	return result, nil
}

// ----------------------------------------------------------------
// プロンプト組み立て
// ----------------------------------------------------------------

func buildSystemPrompt() string {
	return `あなたは家庭料理の提案AIです。ユーザーが持っている食材と条件をもとに、実用的なレシピを提案します。
 
	## 出力ルール
	- 必ずJSON形式のみを返す。説明文・前置き・コードブロック記号(` + "```" + `)は一切含めない
	- 以下のスキーマに厳密に従う
	
	## JSONスキーマ
	{
	"recipes": [
		{
		"title": "料理名",
		"cooking_time_min": 30,
		"genre": "japanese",
		"ingredients": [
			{ "name": "食材名", "quantity": 300, "unit": "g", "from_inventory": true }
		],
		"missing_ingredients": [
			{ "name": "不足食材", "quantity": 5, "unit": "g" }
		],
		"steps": [
			{ "step_no": 1, "description": "手順の説明", "duration_min": 3 }
		]
		}
	]
	}
	
	## フィールド定義
	- genre: "japanese" / "western" / "chinese" / "other" のいずれか
	- from_inventory: 在庫リストに含まれる食材は true、含まれない食材は false
	- missing_ingredients: 在庫にない・不足する食材のみ列挙
	- duration_min: 省略可能（待ち時間がないステップは省略してよい）`
}

// 特定のリクエスト独自のプロンプトを作成
func buildUserPrompt(req *RecommendRequest) string {
	var sb strings.Builder

	// --- 在庫リスト ---
	fmt.Fprint(&sb, "## 現在の在庫\n")
	if len(req.InventoryItems) == 0 {
		fmt.Fprint(&sb, "（在庫なし）\n")
	} else {
		for _, item := range req.InventoryItems {
			if item.ExpiresAt != nil {
				remaining := daysUntil(*item.ExpiresAt)
				switch {
				case remaining < 0:
					fmt.Fprintf(&sb, "（期限切れ %d日前）", -remaining)
				case remaining == 0:
					fmt.Fprint(&sb, "（本日期限）")
				default:
					fmt.Fprintf(&sb, "（期限まで %d日）", remaining)
				}
			}
			fmt.Fprint(&sb, "\n")
		}
	}

	// --- フィルタ条件 ---
	fmt.Fprint(&sb, "\n## 条件\n")
	if req.Filters.MaxCookingTimeMin != nil {
		fmt.Fprintf(&sb, "- 調理時間: %d分以内\n", *req.Filters.MaxCookingTimeMin)
	}

	if req.Filters.Genre != nil {
		genreLabel := map[string]string{
			"japanese": "和食",
			"western":  "洋食",
			"chinese":  "中華",
			"other":    "その他",
		}
		if label, ok := genreLabel[*req.Filters.Genre]; ok {
			fmt.Fprintf(&sb, "- ジャンル: %s\n", label)
		}
	}
	if req.Filters.PreferExpiring {
		fmt.Fprint(&sb, "- 期限が近い食材を優先して使うこと\n")
	}

	// --- 件数 ---
	fmt.Fprintf(&sb, "\n## 提案件数\n%d件\n", req.Count)

	return sb.String()
}

// daysUntil は YYYY-MM-DD 形式の日付文字列から今日までの残日数を返す。
func daysUntil(dateStr string) int {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return 0
	}
	today := time.Now().Truncate(24 * time.Hour)
	return int(t.Truncate(24*time.Hour).Sub(today).Hours() / 24)
}

// ----------------------------------------------------------------
// レスポンスパース
// ----------------------------------------------------------------

func extractText(resp apiResponse) string {
	for _, block := range resp.Content {
		if block.Type == "text" {
			return strings.TrimSpace(block.Text)
		}
	}
	return ""
}

func parseResult(text string) (*model.RecommendationResultPayload, error) {
	// モデルが誤って ``` で囲んだ場合のフォールバック
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)

	var result model.RecommendationResultPayload
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil, fmt.Errorf("json unmarshal failed (raw: %.200s): %w", text, err)
	}
	if len(result.Recipes) == 0 {
		return nil, fmt.Errorf("no recipes in response")
	}
	return &result, nil
}
