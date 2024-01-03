package views

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sngx13/pingernoid/database"
	"github.com/sngx13/pingernoid/models"
)

func WebVisitorsPage(c *gin.Context) {
	userIP := c.MustGet("clientIP").(string)
	c.HTML(
		http.StatusOK,
		"visitors.html",
		gin.H{
			"title":  "Visitors Page",
			"userIP": userIP,
		},
	)
}

func WebDashboardPage(c *gin.Context) {
	userIP := c.MustGet("clientIP").(string)
	c.HTML(
		http.StatusOK,
		"dashboard.html",
		gin.H{
			"title":  "Dashboard Page",
			"userIP": userIP,
		},
	)
}

func WebGetMeasurement(c *gin.Context) {
	userIP := c.MustGet("clientIP").(string)
	msrID := c.Param("id")
	var msr models.PingMeasurement
	var pageErrors error
	if err := database.DB.Preload("Results").Preload("Alerts").Where("id = ?", msrID).First(&msr).Error; err != nil {
		pageErrors = err
	}
	c.HTML(
		http.StatusOK,
		"measurement.html",
		gin.H{
			"title":  "Measurement Page",
			"userIP": userIP,
			"data":   msr,
			"errors": pageErrors,
		},
	)
}
