package scheduler

import (
	"context"
	"log"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/google/uuid"
	"github.com/sngx13/pingernoid/database"
	"github.com/sngx13/pingernoid/models"
	"github.com/sngx13/pingernoid/pinger"
	"github.com/sngx13/pingernoid/utils"
)

func SchedulePingMeasurement(msrID uuid.UUID, target string, count, frequency int) {
	log.Printf("[*] 'SchedulePingMeasurement' - Adding measurement: %s to scheduler...", msrID.String())
	s := gocron.NewScheduler(time.UTC)
	s.WaitForScheduleAll()
	s.Every(frequency).Minutes().Do(func() {
		log.Println("[i] 'SchedulePingMeasurement' - Checking for stopped measurements...")
		var msr models.PingMeasurement
		if err := database.DB.First(&msr, "id = ?", msrID).Error; err != nil {
			log.Printf("[!] 'SchedulePingMeasurement' - Error querying database: %v, could not find measurement: %s", err, msrID.String())
		}
		if msr.Status > utils.StatusStopped && msr.ID != uuid.Nil {
			log.Printf("[i] 'SchedulePingMeasurement' - Measurement: %s is 'RUNNING', performing ICMP test towards: %s", msr.ID.String(), msr.Target)
			pinger.PingIP(context.Background(), msr.ID, msr.Target, msr.PacketCount)
		} else {
			log.Printf("[i] 'SchedulePingMeasurement' - Skipping measurement: %s as it is in 'STOPPED' state.", msr.ID.String())
		}
	})
	s.StartAsync()
}

func SchedulerHouseKeeping() {
	var msrs []models.PingMeasurement
	if err := database.DB.Find(&msrs).Error; err != nil {
		log.Println("[!] 'SchedulerHouseKeeping' - Error querying database:", err)
	}
	for _, msr := range msrs {
		if msr.Status > utils.StatusStopped && msr.ID != uuid.Nil {
			log.Printf("[i] 'SchedulerHouseKeeping' - Resuming polling of measurement: %s after program restart.", msr.ID)
			SchedulePingMeasurement(msr.ID, msr.Target, msr.PacketCount, msr.Frequency)
		}
	}
}
