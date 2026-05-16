package handler

import (
	"github.com/Takapon2512/recipe-recommend/backend/internal/config"
	"github.com/Takapon2512/recipe-recommend/backend/internal/middleware"
	"github.com/Takapon2512/recipe-recommend/backend/internal/repository"
	"github.com/Takapon2512/recipe-recommend/backend/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"net/http"
)

func NewRouter(cfg *config.Config, db *gorm.DB) http.Handler {
	if !cfg.IsDevelopment() {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	r.Use(middleware.Logger())
	r.Use(middleware.CORS(cfg.AllowedOrigins))

	r.GET("/api/health", NewHealthHandler())

	authorized := r.Group("/api")
	authorized.Use(middleware.Auth(cfg))
	{
		// DI組み立て
		userRepo := repository.NewUserRepository(db)
		userService := service.NewUserService(userRepo)
		meHandler := NewMeHandler(userService)

		authorized.GET("/me", meHandler.Get)
		authorized.PATCH("/me", nil)
		authorized.DELETE("/me", nil)

		authorized.GET("/categories", nil)

		authorized.GET("/inventory", nil)
		authorized.POST("/inventory", nil)
		authorized.GET("/inventory/expiring", nil)
		authorized.GET("/inventory/suggest", nil)
		authorized.GET("/inventory/summary", nil)
		authorized.GET("/inventory/:id", nil)
		authorized.PATCH("/inventory/:id", nil)
		authorized.DELETE("/inventory/:id", nil)
		authorized.POST("/inventory/:id/restore", nil)

		authorized.GET("/recipes", nil)
		authorized.POST("/recipes", nil)
		authorized.GET("/recipes/:id", nil)
		authorized.PATCH("/recipes/:id", nil)
		authorized.DELETE("/recipes/:id", nil)

		authorized.GET("/meal-plans", nil)
		authorized.POST("/meal-plans", nil)
		authorized.GET("/meal-plans/:id", nil)
		authorized.PATCH("/meal-plans/:id", nil)
		authorized.DELETE("/meal-plans/:id", nil)

		authorized.POST("/recommendations", nil)
		authorized.GET("/recommendations/:job_id", nil)
	}

	return r
}
