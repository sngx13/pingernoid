package views

import (
	"fmt"
	"net"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sngx13/pingernoid/database"
	"github.com/sngx13/pingernoid/models"
	"github.com/sngx13/pingernoid/scheduler"
	"github.com/sngx13/pingernoid/utils"
)

type AlertDetails struct {
	AvgRtt         float64 `json:"avg_rtt"`
	Jitter         float64 `json:"jitter"`
	Loss           float64 `json:"loss"`
	AlertingASPath string  `json:"alerting_as_path"`
	AlertingIPPath string  `json:"alerting_ip_path"`
	ExpectedASPath string  `json:"expected_as_path"`
	ExpectedIPPath string  `json:"expected_ip_path"`
}

type requestData struct {
	Target      string `json:"target"`
	PacketCount string `json:"packet_count"`
	Frequency   string `json:"frequency"`
}

func ApiGetMeasurements(c *gin.Context) {
	var msrs []models.PingMeasurement
	if err := database.DB.Preload("Results").Preload("Alerts").Find(&msrs).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"status": http.StatusBadRequest, "message": fmt.Sprintf("Error: %s", err)})
		return
	}
	c.IndentedJSON(http.StatusOK, gin.H{
		"status": http.StatusOK,
		"data":   msrs,
	})
}

func ApiGetAlertDetails(c *gin.Context) {
	msrID := c.Param("id")
	timestamp := c.Param("timestamp")
	var msrWithAlert models.PingMeasurement
	var msrLatest models.PingMeasurement
	if err := database.DB.Preload("Results").Where("id = ?", msrID).First(&msrWithAlert).Error; err != nil {
		c.IndentedJSON(http.StatusOK, gin.H{"status": http.StatusBadRequest, "message": fmt.Sprintf("Error: %s", err)})
		return
	}
	if err := database.DB.Preload("Results").Where("id = ?", msrID).Last(&msrLatest).Error; err != nil {
		c.IndentedJSON(http.StatusOK, gin.H{"status": http.StatusBadRequest, "message": fmt.Sprintf("Error: %s", err)})
		return
	}
	latestAsPath := msrLatest.Results[len(msrLatest.Results)-1]
	for _, info := range msrWithAlert.Results {
		if info.Timestamp == timestamp {
			pathsData := AlertDetails{
				AvgRtt:         info.AvgRtt,
				Jitter:         info.Jitter,
				Loss:           info.Loss,
				AlertingASPath: info.ASPath,
				AlertingIPPath: info.IPPath,
				ExpectedASPath: latestAsPath.ASPath,
				ExpectedIPPath: latestAsPath.IPPath,
			}
			c.IndentedJSON(http.StatusOK, gin.H{
				"status": http.StatusOK,
				"data":   pathsData,
			})
			return
		}
	}
	c.IndentedJSON(http.StatusOK, gin.H{"status": http.StatusNotFound, "message": "Could not find requested information"})
}

func ApiGetMeasurement(c *gin.Context) {
	msrID := c.Param("id")
	var msr models.PingMeasurement
	if err := database.DB.Preload("Results").Where("id = ?", msrID).First(&msr).Error; err != nil {
		c.IndentedJSON(http.StatusOK, gin.H{"status": http.StatusBadRequest, "message": fmt.Sprintf("Error: %s", err)})
		return
	}
	c.IndentedJSON(http.StatusOK, gin.H{
		"status": http.StatusOK,
		"data":   msr,
	})
}

func ApiGetMeasurementTracePathGraph(c *gin.Context) {
	msrID := c.Param("id")
	data, err := utils.GenerateTracerouteGraph(uuid.MustParse(msrID))
	if err != nil {
		c.IndentedJSON(http.StatusOK, gin.H{"status": http.StatusBadRequest, "message": fmt.Sprintf("Error: %s", err)})
		return
	}
	c.IndentedJSON(http.StatusOK, gin.H{
		"status": http.StatusOK,
		"data":   data,
	})
}

func ApiGetMeasurementCombinedChartResults(c *gin.Context) {
	msrID := c.Param("id")
	timeRange, err := strconv.Atoi(c.Param("time_range"))
	if err != nil {
		c.IndentedJSON(http.StatusOK, gin.H{"status": http.StatusBadRequest, "message": "Could not convert timeRange to an integer"})
		return
	}
	data, err := utils.GenerateCombinedChartData(msrID, timeRange)
	if err != nil {
		c.IndentedJSON(http.StatusOK, gin.H{
			"status": http.StatusBadRequest,
			"data":   data,
		})
	}
	c.IndentedJSON(http.StatusOK, gin.H{
		"status": http.StatusOK,
		"data":   data,
	})
}

func ApiDeleteMeasurement(c *gin.Context) {
	msrID := c.Param("id")
	_, err := utils.UpdateMsrInDatabase(msrID, utils.StatusNameDelete)
	if err != nil {
		c.IndentedJSON(http.StatusOK, gin.H{"status": http.StatusBadRequest, "message": fmt.Sprintf("Error: %s", err)})
		return
	}
	c.Header("HX-Trigger", "reloadTable")
	message := fmt.Sprintf("Measurement: %s was deleted.", msrID)
	c.IndentedJSON(http.StatusOK, gin.H{"status": http.StatusNoContent, "message": message})
}

func ApiStopMeasurement(c *gin.Context) {
	msrID := c.Param("id")
	msr, err := utils.UpdateMsrInDatabase(msrID, utils.StatusNameStopped)
	if err != nil {
		c.IndentedJSON(http.StatusOK, gin.H{"status": http.StatusBadRequest, "message": fmt.Sprintf("Error: %s", err)})
		return
	}
	c.Header("HX-Trigger", "reloadTable")
	message := fmt.Sprintf("Measurement: %s was stopped.", msrID)
	c.IndentedJSON(http.StatusOK, gin.H{
		"status":  http.StatusAccepted,
		"message": message,
		"data":    msr,
	})
}

func ApiRestartMeasurement(c *gin.Context) {
	msrID := c.Param("id")
	msr, err := utils.UpdateMsrInDatabase(msrID, utils.StatusNameRestarting)
	if err != nil {
		c.IndentedJSON(http.StatusOK, gin.H{"status": http.StatusBadRequest, "message": fmt.Sprintf("Error: %s", err)})
		return
	}
	c.Header("HX-Trigger", "reloadTable")
	message := fmt.Sprintf("Measurement: %s was restarted.", msrID)
	c.IndentedJSON(http.StatusOK, gin.H{
		"status":  http.StatusAccepted,
		"message": message,
		"data":    msr,
	})
}

func ApiCreateMeasurement(c *gin.Context) {
	var requestData requestData
	if err := c.BindJSON(&requestData); err != nil {
		c.IndentedJSON(http.StatusOK, gin.H{"status": http.StatusBadRequest, "message": err.Error()})
		return
	}
	frequency := utils.ConvertStringToInt(requestData.Frequency)
	packetCount := utils.ConvertStringToInt(requestData.PacketCount)
	if requestData.Target == "" || frequency <= 0 || packetCount <= 0 {
		c.IndentedJSON(http.StatusOK, gin.H{"status": http.StatusNotAcceptable, "message": "Missing required parameters"})
		return
	}
	msr, err := utils.AddMsrToDatabase(requestData.Target, packetCount, frequency)
	if err != nil {
		message := fmt.Sprintf("Could not add measurement to database for processing, %v", err)
		c.IndentedJSON(http.StatusOK,
			gin.H{
				"status":  http.StatusBadRequest,
				"message": message,
			})
		return
	}
	scheduler.SchedulePingMeasurement(msr.ID, msr.Target, msr.PacketCount, msr.Frequency)
	c.Header("HX-Trigger", "pageRefresh")
	message := fmt.Sprintf("Measurement: %s was added successfully", msr.ID.String())
	c.IndentedJSON(http.StatusOK,
		gin.H{
			"status":  http.StatusAccepted,
			"message": message,
		})
}

func ApiCheckTargetIP(c *gin.Context) {
	ipAddr := c.PostForm("target")
	targetIP := net.ParseIP(ipAddr)
	if len(ipAddr) >= 7 && len(ipAddr) <= 15 && targetIP != nil {
		if targetIP.IsPrivate() {
			c.IndentedJSON(http.StatusOK, gin.H{"status": http.StatusBadRequest, "message": "Please enter valid public IP address!"})
			return
		} else {
			c.IndentedJSON(http.StatusOK, gin.H{"status": http.StatusOK, "message": "IP is valid!"})
			return
		}
	} else {
		c.IndentedJSON(http.StatusOK, gin.H{"status": http.StatusBadRequest, "message": "Please enter valid IP address!"})
		return
	}
}

func ApiGetVisitorInfo(c *gin.Context) {
	ipAddr := c.Param("ip")
	var user models.SiteVisitor
	if err := database.DB.First(&user, "ip_address = ?", ipAddr).Error; err != nil {
		c.IndentedJSON(http.StatusOK, gin.H{"status": http.StatusBadRequest, "message": fmt.Sprintf("Error: %s", err)})
		return
	}
	c.IndentedJSON(http.StatusOK, gin.H{"status": http.StatusOK, "data": user})
}

func ApiGetVisitorsChart(c *gin.Context) {
	data := utils.GenerateVisitorsChart()
	c.IndentedJSON(http.StatusOK, gin.H{"status": http.StatusOK, "data": data})
}
