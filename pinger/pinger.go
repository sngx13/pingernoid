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

type Alert struct {
	AlertTimestamp string
	AlertReason    string
	AlertMessage   string
}

type PingResult struct {
	Sent   int
	Rcvd   int
	AvgRtt float64
	MinRtt float64
	MaxRtt float64
	Loss   float64
	Jitter float64
}

type TraceResult struct {
	CurrentASPath       string
	CurrentASHopCount   int
	CurrentIPPath       string
	CurrentIPHopCount   int
	CurrentCombinedPath string
	PreviousASPath      string
	PreviousASHopCount  int
	PreviousIPPath      int
	PreviousIPHopCount  int
}

func (p *PingResult) icmpHealthCheck() (bool, Alert) {
	if p.Rcvd != p.Sent {
		log.Printf("[i] 'icmpHealthCheck' - Packets received: %d is not the same as was sent: %d", p.Rcvd, p.Sent)
		alert := Alert{
			AlertTimestamp: time.Now().Format(time.RFC3339),
			AlertReason:    "PACKET_LOSS",
			AlertMessage:   fmt.Sprintf("Packets sent: %d received: %d", p.Sent, p.Rcvd),
		}
		return true, alert
	} else if p.MinRtt > 50 || p.MaxRtt > 500 || p.AvgRtt > 100 {
		log.Printf("[i] 'icmpHealthCheck' - Latency threshold exceeded:RTT (min) > 50ms / (max) > 500ms / (avg) > 100ms - current (min/max/avg): %fms %fms %fms", p.MinRtt, p.MaxRtt, p.AvgRtt)
		alert := Alert{
			AlertTimestamp: time.Now().Format(time.RFC3339),
			AlertReason:    "HIGH_LATENCY",
			AlertMessage:   fmt.Sprintf("Rtt latency threshold reached (Min > 50 / Max > 500 / Avg > 100): Min: %fms, Max: %fms, Avg: %fms", p.MinRtt, p.MaxRtt, p.AvgRtt),
		}
		return true, alert
	} else if p.Jitter > 25 {
		log.Printf("[i] 'icmpHealthCheck' - Jitter threshold exceeded 25ms, current: %fms", p.Jitter)
		alert := Alert{
			AlertTimestamp: time.Now().Format(time.RFC3339),
			AlertReason:    "HIGH_JITTER",
			AlertMessage:   fmt.Sprintf("Jitter threshold exceeded 25ms: %fms", p.Jitter),
		}
		return true, alert
	}
	return false, Alert{}
}

func (t *TraceResult) traceHealthCheck() (bool, Alert) {
	if t.PreviousASHopCount > 1 && t.CurrentASHopCount > t.PreviousASHopCount {
		log.Printf("[i] 'traceHealthCheck' - Current AS path: %s is longer than previous one: %s", t.CurrentASPath, t.PreviousASPath)
		alert := Alert{
			AlertTimestamp: time.Now().Format(time.RFC3339),
			AlertReason:    "AS_PATH_CHANGE_LONGER",
			AlertMessage:   fmt.Sprintf("AS Hop count has changed - current: %d, previous: %d", t.CurrentASHopCount, t.PreviousASHopCount),
		}
		return true, alert
	} else if t.PreviousASHopCount > 1 && t.PreviousASHopCount > t.CurrentASHopCount {
		log.Printf("[i] 'traceHealthCheck' - Current AS path: %s is shorter than previous one: %s", t.CurrentASPath, t.PreviousASPath)
		alert := Alert{
			AlertTimestamp: time.Now().Format(time.RFC3339),
			AlertReason:    "AS_PATH_CHANGE_SHORTER",
			AlertMessage:   fmt.Sprintf("AS Hop count has changed - current: %d, previous: %d", t.CurrentASHopCount, t.PreviousASHopCount),
		}
		return true, alert
	} else if t.PreviousIPHopCount > 1 && t.CurrentIPHopCount > t.PreviousIPHopCount {
		log.Printf("[i] 'traceHealthCheck' - Current IP path: %s is longer than previous one: %s", t.CurrentASPath, t.PreviousASPath)
		alert := Alert{
			AlertTimestamp: time.Now().Format(time.RFC3339),
			AlertReason:    "IP_PATH_CHANGE_LONGER",
			AlertMessage:   fmt.Sprintf("IP Hop count has changed - current: %d, previous: %d", t.CurrentIPHopCount, t.PreviousIPHopCount),
		}
		return true, alert
	} else if t.PreviousIPHopCount > 1 && t.PreviousIPHopCount > t.CurrentIPHopCount {
		log.Printf("[i] 'traceHealthCheck' - Current IP path: %s is shorter than previous one: %s", t.CurrentASPath, t.PreviousASPath)
		alert := Alert{
			AlertTimestamp: time.Now().Format(time.RFC3339),
			AlertReason:    "IP_PATH_CHANGE_SHORTER",
			AlertMessage:   fmt.Sprintf("IP Hop count has changed - current: %d, previous: %d", t.CurrentIPHopCount, t.PreviousIPHopCount),
		}
		return true, alert
	}
	return false, Alert{}
}

func saveResult(msrID uuid.UUID, target string, pingResult PingResult, traceResult TraceResult) error {
	var pingMsr models.PingMeasurement
	if err := database.DB.Where("id = ?", msrID).First(&pingMsr).Error; err != nil {
		log.Println("[!] 'saveResult' - Error loading existing measurement:", err)
		return err
	}
	newResults := models.MeasurementResults{
		MsrID:        msrID,
		Timestamp:    time.Now().Format(time.RFC3339),
		Rcvd:         pingResult.Rcvd,
		Sent:         pingResult.Sent,
		Loss:         pingResult.Loss,
		AvgRtt:       pingResult.AvgRtt,
		MinRtt:       pingResult.MinRtt,
		MaxRtt:       pingResult.MaxRtt,
		Jitter:       pingResult.Jitter,
		IPHopCount:   traceResult.CurrentIPHopCount,
		ASHopCount:   traceResult.CurrentASHopCount,
		IPPath:       traceResult.CurrentIPPath,
		ASPath:       traceResult.CurrentASPath,
		CombinedPath: traceResult.CurrentCombinedPath,
	}
	pingMsr.LastPollAt = time.Now().Format(time.RFC3339)
	pingMsr.Status = utils.StatusRunning
	pingMsr.StatusName = utils.StatusNameRunning
	// Alerting
	pingAlert, alertInfo := pingResult.icmpHealthCheck()
	if pingAlert {
		newResults.Alerting = true
		newAlert := models.MeasurementResultAlerts{
			AlertTimestamp: alertInfo.AlertTimestamp,
			AlertReason:    alertInfo.AlertReason,
			AlertMessage:   alertInfo.AlertMessage,
		}
		pingMsr.Alerts = append(pingMsr.Alerts, newAlert)
	}
	traceAlert, alertInfo := traceResult.traceHealthCheck()
	if traceAlert {
		newResults.Alerting = true
		newAlert := models.MeasurementResultAlerts{
			AlertTimestamp: alertInfo.AlertTimestamp,
			AlertReason:    alertInfo.AlertReason,
			AlertMessage:   alertInfo.AlertMessage,
		}
		pingMsr.Alerts = append(pingMsr.Alerts, newAlert)
	}
	pingMsr.Results = append(pingMsr.Results, newResults)
	if err := database.DB.Save(&pingMsr).Error; err != nil {
		log.Println("[!] 'saveResult' - Error updating measurement:", err)
		return err
	}
	return nil
}

func traceIP(msrID uuid.UUID, target string) TraceResult {
	var ipPath, asPath, combinedPath []string
	hops, err := traceroute.Trace(net.ParseIP(target))
	if err != nil {
		log.Println("[!] 'tracePath' - Problem performing traceroute:", err)
		return TraceResult{}
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
	asPath = utils.RemoveDuplicates(asPath)
	previousPaths, err := utils.GetPreviousMsrResult(msrID)
	if err != nil {
		log.Println("[!] 'tracePath' - Could not get previous result", err)
	}
	traceResult := TraceResult{
		CurrentASPath:       strings.Join(asPath, " > "),
		CurrentASHopCount:   len(asPath),
		CurrentIPPath:       strings.Join(ipPath, " > "),
		CurrentIPHopCount:   len(ipPath),
		CurrentCombinedPath: strings.Join(combinedPath, " > "),
		PreviousASPath:      previousPaths.ASPath,
		PreviousASHopCount:  len(strings.Split(previousPaths.ASPath, ">")),
		PreviousIPHopCount:  len(strings.Split(previousPaths.IPPath, ">")),
	}
	return traceResult
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
	pingResult := PingResult{
		Sent:   pinger.Statistics().PacketsSent,
		Rcvd:   pinger.Statistics().PacketsRecv,
		Loss:   pinger.Statistics().PacketLoss,
		AvgRtt: float64(pinger.Statistics().AvgRtt.Milliseconds()),
		MinRtt: float64(pinger.Statistics().MinRtt.Milliseconds()),
		MaxRtt: float64(pinger.Statistics().MaxRtt.Milliseconds()),
		Jitter: float64(pinger.Statistics().StdDevRtt.Milliseconds()),
	}
	traceResult := traceIP(msrID, target)
	if err := saveResult(msrID, target, pingResult, traceResult); err != nil {
		log.Println("[!] 'PingIP' - Attempt to save measurement results failed.")
		return err
	}
	log.Printf("[i] ICMP Statistics for target: %s \n%+v", target, pingResult)
	return nil
}
