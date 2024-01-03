package pinger

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pixelbender/go-traceroute/traceroute"
	probing "github.com/prometheus-community/pro-bing"
	"github.com/sngx13/pingernoid/database"
	"github.com/sngx13/pingernoid/models"
	"github.com/sngx13/pingernoid/utils"
)

// ICMP
var (
	icmpTimeout = 5
	icmpTTL     = 64
)

func removeAsPathDuplicates(elements []string) []string {
	encountered := make(map[string]bool)
	result := []string{}

	for _, v := range elements {
		if !encountered[v] {
			encountered[v] = true
			result = append(result, v)
		}
	}
	return result
}

func tracePath(msrID uuid.UUID, target string) (string, string, string, string, int, int, int, int, bool) {
	var alert bool
	var alertType string
	var ipPath []string
	var asPath []string
	var combinedPath []string
	hops, err := traceroute.Trace(net.ParseIP(target))
	if err != nil {
		log.Println("[!] 'tracePath' - Problem performing traceroute:", err)
		return "N/A", "N/A", "N/A", "N/A", 0, 0, 0, 0, false
	}
	for _, h := range hops {
		for _, n := range h.Nodes {
			hopIP := net.IP.String(n.IP)
			_, asn, _, _ := utils.IPAddrLookupInfo(hopIP)
			ipPath = append(ipPath, hopIP)
			asPath = append(asPath, asn)
			combinedPath = append(combinedPath, fmt.Sprintf("%s (%s)", hopIP, asn))
		}
	}
	asPath = removeAsPathDuplicates(asPath)
	currentIpPath := strings.Join(ipPath, " > ")
	currentAsPath := strings.Join(asPath, " > ")
	currentCombinedPath := strings.Join(combinedPath, " > ")
	previousPaths, err := utils.GetPreviousMsrResult(msrID)
	if err != nil {
		log.Println("[!] 'tracePath' - Could not get previous result", err)
		alert = false
	}
	previousIpHopCount := len(strings.Split(previousPaths.IPPath, ">"))
	previousAsHopCount := len(strings.Split(previousPaths.ASPath, ">"))
	currentIpHopCount := len(ipPath)
	currentAsHopCount := len(asPath)
	if previousIpHopCount > 1 && currentIpPath == previousPaths.IPPath {
		log.Printf("[i] 'tracePath' - Current IP path: %s is identical to previous one", currentIpPath)
		alert = false
	} else if previousIpHopCount > 1 && currentIpPath != previousPaths.IPPath {
		log.Printf("[i] 'tracePath' - Current IP path: %s is not identical to previous one: %s", currentIpPath, previousPaths.IPPath)
		alert = true
		alertType = "IP_PATH_CHANGE"
	}
	if previousAsHopCount > 1 && currentAsPath == previousPaths.ASPath {
		log.Printf("[i] 'tracePath' - Current AS path: %s is identical to previous one", currentAsPath)
		alert = false
	} else if previousAsHopCount > 1 && currentAsPath != previousPaths.ASPath {
		log.Printf("[i] 'tracePath' - Current AS path: %s is not identical to previous one: %s", currentAsPath, previousPaths.ASPath)
		alert = true
		alertType = "AS_PATH_CHANGE"
	}
	return currentIpPath, currentAsPath, currentCombinedPath, alertType, currentIpHopCount, currentAsHopCount, previousIpHopCount, previousAsHopCount, alert
}

func saveResult(msrID uuid.UUID, stats *probing.Statistics, target string) error {
	var pingMsr models.PingMeasurement
	if err := database.DB.Where("id = ?", msrID).First(&pingMsr).Error; err != nil {
		log.Println("[!] 'saveResult' - Error loading existing measurement:", err)
		return err
	}
	currentIpPath, currentAsPath, currentCombinedPath, alertType, currentIpHopCount, currentAsHopCount, previousIpHopCount, previousAsHopCount, alert := tracePath(msrID, target)
	newResults := models.MeasurementResults{
		MsrID:        msrID,
		Timestamp:    time.Now().Format(time.RFC3339),
		Rcvd:         stats.PacketsRecv,
		Sent:         stats.PacketsSent,
		Loss:         stats.PacketLoss,
		AvgRtt:       float64(stats.AvgRtt.Milliseconds()),
		MinRtt:       float64(stats.MinRtt.Milliseconds()),
		MaxRtt:       float64(stats.MaxRtt.Milliseconds()),
		Jitter:       float64(stats.StdDevRtt.Milliseconds()),
		IPHopCount:   currentIpHopCount,
		ASHopCount:   currentAsHopCount,
		IPPath:       currentIpPath,
		ASPath:       currentAsPath,
		CombinedPath: currentCombinedPath,
	}
	pingMsr.LastPollAt = time.Now().Format(time.RFC3339)
	pingMsr.Status = utils.StatusRunning
	pingMsr.StatusName = utils.StatusNameRunning
	pingMsr.Results = append(pingMsr.Results, newResults)
	// Alerting
	if alert {
		var alertMessage string
		if strings.HasPrefix(alertType, "IP") {
			alertMessage = fmt.Sprintf("IP Hop count has changed - current: %d, previous: %d", currentIpHopCount, previousIpHopCount)
		} else if strings.HasPrefix(alertType, "AS") {
			alertMessage = fmt.Sprintf("AS Hop count has changed - current: %d, previous: %d", currentAsHopCount, previousAsHopCount)
		}
		newAlert := models.MeasurementResultAlerts{
			AlertTimestamp: time.Now().Format(time.RFC3339),
			AlertReason:    alertType,
			AlertMessage:   alertMessage,
		}
		pingMsr.Alerts = append(pingMsr.Alerts, newAlert)
	}
	if stats.PacketLoss > 0 {
		newAlert := models.MeasurementResultAlerts{
			AlertTimestamp: time.Now().Format(time.RFC3339),
			AlertReason:    alertType,
			AlertMessage:   fmt.Sprintf("Packets received: %d is not the same as was sent: %d - currect loss: %f", stats.PacketsRecv, stats.PacketsSent, stats.PacketLoss),
		}
		pingMsr.Alerts = append(pingMsr.Alerts, newAlert)
	}
	if float64(stats.MinRtt.Milliseconds()) > 50 || float64(stats.AvgRtt.Milliseconds()) > 100 || float64(stats.MaxRtt.Milliseconds()) > 500 {
		newAlert := models.MeasurementResultAlerts{
			AlertTimestamp: time.Now().Format(time.RFC3339),
			AlertReason:    "LATENCY_THRESHOLD",
			AlertMessage:   fmt.Sprintf("Latency threshold exceeded:RTT (min) > 50ms / (avg) > 100ms / (max) > 500ms - current (avg): %dms", stats.AvgRtt.Milliseconds()),
		}
		pingMsr.Alerts = append(pingMsr.Alerts, newAlert)
	}
	if err := database.DB.Save(&pingMsr).Error; err != nil {
		log.Println("[!] 'saveResult' - Error updating measurement:", err)
		return err
	}
	return nil
}

func PingIP(ctx context.Context, msrID uuid.UUID, target string, count int) error {
	if count >= icmpTimeout && count <= 100 {
		icmpTimeout = count
	} else if count <= icmpTimeout {
		icmpTimeout = count
	} else {
		return fmt.Errorf("requested count: %d is not supported, value should be less than 100 and more than 0", count)
	}
	pinger, err := probing.NewPinger(target)
	pinger.Count = count
	pinger.Timeout = time.Duration(icmpTimeout) * time.Second
	pinger.TTL = icmpTTL
	if err != nil {
		log.Println("[!] 'PingIP' - Error:", err)
		return err
	}
	if err := pinger.RunWithContext(ctx); err != nil {
		log.Println("[!] 'PingIP' - There has been a problem with sending ICMP packets to the target.")
		return err
	}
	if err := saveResult(msrID, pinger.Statistics(), target); err != nil {
		log.Println("[!] 'PingIP' - Attempt to save measurement results failed.")
		return err
	}
	log.Printf(
		"[i] ICMP Statistics for target: %s -> Sent: %d , Rcvd: %d, Loss: %f%%, Latency (Avg): %dms",
		pinger.Statistics().IPAddr,
		pinger.Statistics().PacketsSent,
		pinger.Statistics().PacketsRecv,
		pinger.Statistics().PacketLoss,
		pinger.Statistics().AvgRtt.Milliseconds(),
	)
	return nil
}
