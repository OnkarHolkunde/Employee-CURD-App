package routes

import (
	"net/http"

	"excel-crud-app/internal/database"
	"excel-crud-app/internal/handlers"
	"excel-crud-app/internal/middleware"

	"github.com/gin-gonic/gin"
)

// NewRouter builds the fully configured Gin engine: global middleware,
// health/readiness checks, and versioned API routes.
func NewRouter(allowedOrigin string, uploadH *handlers.UploadHandler, employeeH *handlers.EmployeeHandler) *gin.Engine {
	router := gin.New()

	// Order matters: recover first, then tag the request, then log it.
	router.Use(middleware.Recovery())
	router.Use(middleware.RequestID())
	router.Use(middleware.StructuredLogger())
	router.Use(middleware.CORS(allowedOrigin))

	router.GET("/health", livenessCheck)
	router.GET("/ready", readinessCheck)

	v1 := router.Group("/api/v1")
	{
		v1.POST("/upload", uploadH.UploadExcel)
		v1.GET("/upload/status/:job_id", uploadH.GetUploadStatus)

		employees := v1.Group("/employees")
		{
			employees.GET("", employeeH.ListEmployees)
			employees.GET("/:id", employeeH.GetEmployee)
			employees.POST("", employeeH.CreateEmployee)
			employees.PUT("/:id", employeeH.ReplaceEmployee)
			employees.PATCH("/:id", employeeH.PatchEmployee)
			employees.DELETE("/:id", employeeH.DeleteEmployee)
		}
	}

	return router
}

// livenessCheck just confirms the process is up and serving requests.
func livenessCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// readinessCheck additionally confirms MySQL and Redis are reachable —
// what a load balancer should actually gate traffic on.
func readinessCheck(c *gin.Context) {
	problems := gin.H{}

	if sqlDB, err := database.DB.DB(); err != nil || sqlDB.PingContext(c.Request.Context()) != nil {
		problems["mysql"] = "unreachable"
	}
	if err := database.RDB.Ping(c.Request.Context()).Err(); err != nil {
		problems["redis"] = "unreachable"
	}

	if len(problems) > 0 {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready", "problems": problems})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}
