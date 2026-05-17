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

	stub := func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{
			"error": gin.H{"code": "NOT_IMPLEMENTED", "message": "not implemented"},
		})
	}

	r.GET("/api/health", NewHealthHandler())

	authorized := r.Group("/api")
	authorized.Use(middleware.Auth(cfg))
	{
		// DI組み立て
		userRepo := repository.NewUserRepository(db)
		userService := service.NewUserService(userRepo)
		meHandler := NewMeHandler(userService)

		categoryRepo := repository.NewCategoryRepository(db)
		categorySvc := service.NewCategoryService(categoryRepo)
		categoryHandler := NewCategoryHandler(categorySvc)

		authorized.GET("/me", meHandler.Get)
		authorized.PATCH("/me", meHandler.Patch)
		authorized.DELETE("/me", meHandler.Delete)

		authorized.GET("/categories", categoryHandler.List)

		authorized.GET("/inventory", stub)
		authorized.POST("/inventory", stub)
		authorized.GET("/inventory/expiring", stub)
		authorized.GET("/inventory/suggest", stub)
		authorized.GET("/inventory/summary", stub)
		authorized.GET("/inventory/:id", stub)
		authorized.PATCH("/inventory/:id", stub)
		authorized.DELETE("/inventory/:id", stub)
		authorized.POST("/inventory/:id/restore", stub)

		authorized.GET("/recipes", stub)
		authorized.POST("/recipes", stub)
		authorized.GET("/recipes/:id", stub)
		authorized.PATCH("/recipes/:id", stub)
		authorized.DELETE("/recipes/:id", stub)

		authorized.GET("/meal-plans", stub)
		authorized.POST("/meal-plans", stub)
		authorized.GET("/meal-plans/:id", stub)
		authorized.PATCH("/meal-plans/:id", stub)
		authorized.DELETE("/meal-plans/:id", stub)

		authorized.POST("/recommendations", stub)
		authorized.GET("/recommendations/:job_id", stub)
	}

	return r
}
