package cognito

import (
	"context"
	"fmt"

	"github.com/Takapon2512/recipe-recommend/backend/internal/config"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

type Claims struct {
	Sub      string
	Email    string
	Provider string
}

type Verifier struct {
	cache    *jwk.Cache
	issuer   string
	clientID string
}

func NewVerifier(cfg *config.Config) *Verifier {
	issuer := fmt.Sprintf(
		"https://cognito-idp.%s.amazonaws.com/%s",
		cfg.CognitoRegion,
		cfg.CognitoUserPoolID,
	)

	jwksURL := issuer + "/.well-known/jwks.json"

	cache := jwk.NewCache(context.Background())
	if err := cache.Register(jwksURL); err != nil {
		panic(fmt.Errorf("JWKsキャッシュ登録失敗: %w", err))
	}

	return &Verifier{
		cache:    cache,
		issuer:   issuer,
		clientID: cfg.CognitoClientID,
	}
}

func (v *Verifier) Verify(ctx context.Context, tokenStr string) (*Claims, error) {
	jwksURL := v.issuer + "/.well-known/jwks.json"

	keySet, err := v.cache.Get(ctx, jwksURL)
	if err != nil {
		return nil, fmt.Errorf("JWKs取得失敗: %w", err)
	}

	token, err := jwt.Parse([]byte(tokenStr),
		jwt.WithKeySet(keySet),
		jwt.WithValidate(true),
		jwt.WithIssuer(v.issuer),
		jwt.WithAudience(v.clientID),
	)
	if err != nil {
		return nil, fmt.Errorf("JWT検証失敗: %w", err)
	}

	sub := token.Subject()
	emailRaw, _ := token.Get("email")
	email, _ := emailRaw.(string)

	providerRaw, _ := token.Get("custom:provider")
	provider, _ := providerRaw.(string)

	return &Claims{Sub: sub, Email: email, Provider: provider}, nil
}
