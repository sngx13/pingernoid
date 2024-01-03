package main

import (
	"log"
	"net"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sngx13/pingernoid/database"
	"github.com/sngx13/pingernoid/models"
	"github.com/sngx13/pingernoid/scheduler"
	"github.com/sngx13/pingernoid/utils"
	"github.com/sngx13/pingernoid/views"
	"gorm.io/gorm"
)

const (
	LOCALHOST_443 = "0.0.0.0:443"
)

func ClientIPMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the client's real IP address
		ip := c.ClientIP()
		// Store the IP address in the context for later use
		c.Set("clientIP", ip)
		// Create a user and add it to the database if IP is globally unique (Public)
		userIP := net.ParseIP(ip)
		if userIP != nil && !userIP.IsPrivate() {
			var user models.SiteVisitor
			if err := database.DB.First(&user, "ip_address = ?", ip).Error; err != nil {
				log.Printf("[i] User: %s has not visited us before, adding to the database.", ip)
				isp, asn, country, countryCode := utils.IPAddrLookupInfo(ip)
				user := &models.SiteVisitor{
					IPAddress:   ip,
					ISP:         isp,
					ASN:         asn,
					Country:     country,
					CountryCode: countryCode,
				}
				db.Create(&user)
			}
		}
		// Call the next handler
		c.Next()
	}
}

func main() {
	// Database
	log.Println("[i] Performing database initialisation and model migrations.")
	database.DBInit()
	err := database.DB.AutoMigrate(
		&models.PingMeasurement{},
		&models.MeasurementResults{},
		&models.MeasurementResultAlerts{},
		&models.SiteVisitor{},
	)
	if err != nil {
		log.Println("[!] Database migration error:", err)
	}
	// Housekeeping
	scheduler.SchedulerHouseKeeping()
	// Gin Router
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	router.Use(gin.Recovery())
	// Use the custom middleware to extract and store the IP address
	router.Use(ClientIPMiddleware(database.DB))
	// Use custom delims to prevent clashes with HTMX
	router.Delims("{[{", "}]}")
	// Templates
	router.LoadHTMLGlob("templates/**/*")
	// Static
	router.Static("/static", "./static")
	// API Endpoints
	api_v1 := router.Group("/api/v1")
	api_v1.POST("/checks/target/verify", views.ApiCheckTargetIP)
	api_v1.GET("/measurements", views.ApiGetMeasurements)
	api_v1.GET("/measurements/:id", views.ApiGetMeasurement)
	api_v1.GET("/measurements/:id/traceroute/path", views.ApiGetMeasurementTracePathGraph)
	api_v1.POST("/measurements/create", views.ApiCreateMeasurement)
	api_v1.POST("/measurements/:id/stop", views.ApiStopMeasurement)
	api_v1.POST("/measurements/:id/restart", views.ApiRestartMeasurement)
	api_v1.DELETE("/measurements/:id/delete", views.ApiDeleteMeasurement)
	api_v1.GET("/measurements/:id/results/combined/:time_range", views.ApiGetMeasurementCombinedChartResults)
	api_v1.GET("/site/visitor/info/:ip", views.ApiGetVisitorInfo)
	api_v1.GET("/site/visitor/info/chart", views.ApiGetVisitorsChart)
	// WEB Endpoints
	web_v1 := router.Group("/")
	web_v1.GET("/", views.WebDashboardPage)
	web_v1.GET("/visitors", views.WebVisitorsPage)
	web_v1.GET("/measurement/:id", views.WebGetMeasurement)
	router.NoRoute(func(c *gin.Context) {
		c.Redirect(http.StatusPermanentRedirect, "/")
	})
	// Run HTTP Server
	certPath := "/etc/letsencrypt/live/sngx-mrqbpbkwmk.dynamic-m.com/"
	router.RunTLS(LOCALHOST_443, certPath+"fullchain.pem", certPath+"privkey.pem")
}
