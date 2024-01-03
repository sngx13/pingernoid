package utils

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/sngx13/pingernoid/database"
	"github.com/sngx13/pingernoid/models"
)

// States
const (
	StatusDelete         = -1
	StatusNameDelete     = "DELETE"
	StatusStopped        = 0
	StatusNameStopped    = "STOPPED"
	StatusRunning        = 1
	StatusNameRunning    = "RUNNING"
	StatusScheduled      = 2
	StatusNameScheduled  = "SCHEDULED"
	StatusRestarting     = 3
	StatusNameRestarting = "RESTARTING"
)

type CountryVisitorsCount struct {
	Country string
	Count   int64
}

type RttData struct {
	X time.Time `json:"x"`
	Y float64   `json:"y"`
}

type IPLookupData struct {
	Status      string  `json:"status"`
	Country     string  `json:"country"`
	CountryCode string  `json:"countryCode"`
	Region      string  `json:"region"`
	RegionName  string  `json:"regionName"`
	City        string  `json:"city"`
	Zip         string  `json:"zip"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	Timezone    string  `json:"timezone"`
	Isp         string  `json:"isp"`
	Org         string  `json:"org"`
	As          string  `json:"as"`
	Query       string  `json:"query"`
}

func GenerateUUID() uuid.UUID {
	uuid, err := uuid.NewUUID()
	if err != nil {
		log.Println("[!] 'GenerateUUID' - Could not generate UUID:", err)
	}
	return uuid
}

func checkIfTargetAlreadyExists(target string) (bool, error) {
	var msr models.PingMeasurement
	if err := database.DB.First(&msr, "target = ?", target).Error; err != nil {
		return false, err
	}
	return true, errors.Errorf("Measurement with target IP: %s already exists with ID: %s", target, msr.ID.String())
}

func UpdateMsrInDatabase(msrID string, stateChange string) (models.PingMeasurement, error) {
	var msr models.PingMeasurement
	if err := database.DB.First(&msr, "id = ?", msrID).Error; err != nil {
		return msr, err
	}
	switch stateChange {
	case StatusNameStopped:
		log.Printf("[i] 'UpdateMsrInDatabase' - Received request to 'STOP' measurement: %s", msrID)
		msr.StoppedAt = time.Now().Format(time.RFC3339)
		msr.Status = StatusStopped
		msr.StatusName = StatusNameStopped
		if err := database.DB.Save(&msr).Error; err != nil {
			return msr, err
		}
	case StatusNameRestarting:
		log.Printf("[i] 'UpdateMsrInDatabase' - Received request to 'RESTART' measurement: %s", msrID)
		msr.StoppedAt = ""
		msr.Status = StatusRestarting
		msr.StatusName = StatusNameRestarting
		if err := database.DB.Save(&msr).Error; err != nil {
			return msr, err
		}
	case StatusNameDelete:
		log.Printf("[i] 'UpdateMsrInDatabase' - Received request to 'DELETE' measurement: %s", msrID)
		var msr models.PingMeasurement
		var msrResults models.MeasurementResults
		var msrAlerts models.MeasurementResultAlerts
		if err := database.DB.Where("id = ?", msrID).Delete(&msr).Error; err != nil {
			return msr, err
		}
		if err := database.DB.Where("msr_id = ?", msrID).Delete(&msrResults).Error; err != nil {
			return msr, err
		}
		if err := database.DB.Where("msr_id = ?", msrID).Delete(&msrAlerts).Error; err != nil {
			return msr, err
		}
	}
	return msr, nil
}

func AddMsrToDatabase(target string, count, frequency int) (models.PingMeasurement, error) {
	log.Println("[i] 'AddMsrToDatabase' - Attempting to add measurement to the database.")
	exists, err := checkIfTargetAlreadyExists(target)
	if !exists {
		var isHostname bool
		if net.ParseIP(target) != nil {
			isHostname = false
		} else {
			isHostname = false
		}
		data := models.PingMeasurement{
			ID:          GenerateUUID(),
			CreatedAt:   time.Now().Format(time.RFC3339),
			Target:      target,
			IsHostname:  isHostname,
			Frequency:   frequency,
			PacketCount: count,
			Status:      StatusScheduled,
			StatusName:  StatusNameScheduled,
			LastPollAt:  "Never",
		}
		if err := database.DB.Create(&data).Error; err != nil {
			log.Println("[!] 'AddMsrToDatabase' - There has been a problem with adding measurement to the database.", err)
			return models.PingMeasurement{}, errors.Wrap(err, "Problem saving measurement to database.")
		}
		log.Println("[i] 'AddMsrToDatabase' - Measurement was added to database with id:", data.ID)
		return data, nil
	} else {
		return models.PingMeasurement{}, err
	}
}

func GetPreviousMsrResult(msrID uuid.UUID) (models.MeasurementResults, error) {
	var result models.MeasurementResults
	if err := database.DB.Last(&result, "msr_id = ?", msrID).Error; err != nil {
		log.Println("[!] 'GetPreviousMsrResult' - There has been a problem finding previous results", err)
		return result, err
	}
	return result, nil
}

func ConvertStringToInt(object string) int {
	intObject, err := strconv.Atoi(object)
	if err != nil {
		return 0
	} else {
		return intObject
	}
}

func getResultsInTimeRange(results models.PingMeasurement, timeRange int) []models.MeasurementResults {
	requestedNumberOfResults := 60 / results.Frequency * timeRange
	if requestedNumberOfResults > len(results.Results) {
		log.Printf("[!] 'getResultsInTimeRange' - Number of requested results: %d is larger than stored: %d, returning all we have...", requestedNumberOfResults, len(results.Results))
		return results.Results
	}
	log.Printf("[i] 'getResultsInTimeRange' - Returning: %d worth of measurement results", requestedNumberOfResults)
	return results.Results[len(results.Results)-requestedNumberOfResults:]
}

func populateResultSlices(resultsInTimeRange []models.MeasurementResults) ([]RttData, []RttData, []RttData, []RttData, []RttData, []RttData, []RttData, []RttData, []RttData) {
	var (
		rttMinResults, rttMaxResults, rttAvgResults, jitterResults, pktSentResults, pktRcvdResults, pktLossResults, ipHopCountResults, asHopCountResults []RttData
	)
	for _, result := range resultsInTimeRange {
		timestamp, err := time.Parse(time.RFC3339, result.Timestamp)
		if err != nil {
			log.Printf("[!] 'populateResultSlices' - Could not convert: %s to 'time.Time'", result.Timestamp)
			continue
		}
		rttMinResults = append(rttMinResults, RttData{X: timestamp, Y: result.MinRtt})
		rttMaxResults = append(rttMaxResults, RttData{X: timestamp, Y: result.MaxRtt})
		rttAvgResults = append(rttAvgResults, RttData{X: timestamp, Y: result.AvgRtt})
		jitterResults = append(jitterResults, RttData{X: timestamp, Y: result.Jitter})
		pktSentResults = append(pktSentResults, RttData{X: timestamp, Y: float64(result.Sent)})
		pktRcvdResults = append(pktRcvdResults, RttData{X: timestamp, Y: float64(result.Rcvd)})
		pktLossResults = append(pktLossResults, RttData{X: timestamp, Y: float64(result.Loss)})
		ipHopCountResults = append(ipHopCountResults, RttData{X: timestamp, Y: float64(result.IPHopCount)})
		asHopCountResults = append(asHopCountResults, RttData{X: timestamp, Y: float64(result.ASHopCount)})
	}
	return rttMinResults, rttMaxResults, rttAvgResults, jitterResults, pktSentResults, pktRcvdResults, pktLossResults, ipHopCountResults, asHopCountResults
}

func createResponseMap(name string, data []RttData) map[string]interface{} {
	return map[string]interface{}{
		"name": name,
		"data": data,
	}
}

func GenerateCombinedChartData(msrID string, timeRange int) (map[string]map[string]interface{}, error) {
	data := make(map[string]map[string]interface{})
	var results models.PingMeasurement
	if err := database.DB.Preload("Results").First(&results, "id = ?", msrID).Error; err != nil {
		return data, err
	}
	resultsInTimeRange := getResultsInTimeRange(results, timeRange)
	rttMinResults, rttMaxResults, rttAvgResults, jitterResults, pktSentResults, pktRcvdResults, pktLossResults, ipHopCountResults, asHopCountResults := populateResultSlices(resultsInTimeRange)
	data["Rtt"] = map[string]interface{}{
		"Jitter":     createResponseMap("Jitter", jitterResults),
		"LatencyAvg": createResponseMap("Latency (Avg)", rttAvgResults),
		"LatencyMax": createResponseMap("Latency (Max)", rttMaxResults),
		"LatencyMin": createResponseMap("Latency (Min)", rttMinResults),
	}
	data["Pkt"] = map[string]interface{}{
		"PacketsSent": createResponseMap("Packets Sent", pktSentResults),
		"PacketsRcvd": createResponseMap("Packets Rcvd", pktRcvdResults),
		"PacketsLost": createResponseMap("Packets Lost", pktLossResults),
	}
	data["Hop"] = map[string]interface{}{
		"IPHopCount": createResponseMap("IP Hops", ipHopCountResults),
		"ASHopCount": createResponseMap("AS Hops", asHopCountResults),
	}
	return data, nil
}

func IPAddrLookupInfo(ipAddr string) (string, string, string, string) {
	isp, asn, country, countryCode := "", "", "", ""
	asnURL := "http://ip-api.com/json/" + ipAddr
	resp, err := http.Get(asnURL)
	if err != nil {
		log.Println("[!] 'IPAddrLookupInfo' - Could not complete user IP lookup request:", err)
		return "", "", "", ""
	}
	defer resp.Body.Close()
	ipInfo := IPLookupData{}
	if err := json.NewDecoder(resp.Body).Decode(&ipInfo); err != nil {
		log.Println("[!] 'IPAddrLookupInfo' - Could not decode response from API:", err)
		return "", "", "", ""
	} else {
		isp, asn, country, countryCode = ipInfo.Isp, ipInfo.As, ipInfo.Country, ipInfo.CountryCode
	}
	return isp, asn, country, countryCode
}

func GenerateVisitorsChart() []map[string]interface{} {
	var visitors []models.SiteVisitor
	var countryVisitorsCount []CountryVisitorsCount
	database.DB.Model(&visitors).
		Select("country, country_code, COUNT(*) as count").
		Group("country, country_code").
		Order("country, country_code").
		Find(&countryVisitorsCount)
	var chart []map[string]interface{}
	for _, v := range countryVisitorsCount {
		chartData := map[string]interface{}{
			"X": v.Country,
			"Y": v.Count,
		}
		chart = append(chart, chartData)
	}
	return chart
}

func GenerateTracerouteGraph(msrID uuid.UUID) (map[string][]interface{}, error) {
	nodeData := make(map[string][]interface{})
	var edgeData []interface{}
	previousPathTaken, err := GetPreviousMsrResult(msrID)
	if err != nil {
		return nil, err
	}
	ipPathHops := strings.Split(previousPathTaken.CombinedPath, ">")
	for index, ipHop := range ipPathHops {
		hopData := map[string]any{
			"id":    index,
			"label": ipHop,
		}
		nodeData["nodes"] = append(nodeData["nodes"], hopData)
		if index+1 < len(ipPathHops) {
			edge := map[string]int{
				"from": index,
				"to":   index + 1,
			}
			edgeData = append(edgeData, edge)
		}
	}
	nodeData["edges"] = edgeData
	return nodeData, nil
}
