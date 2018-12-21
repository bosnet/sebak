package common

import (
	"bytes"
	"fmt"
	"os/exec"
	"time"

	"github.com/beevik/ntp"
)

const (
	TIMEFORMAT_ISO8601 string = "2006-01-02T15:04:05.000000000Z07:00"
)

func FormatISO8601(t time.Time) string {
	return t.Format(TIMEFORMAT_ISO8601)
}

func NowISO8601() string {
	return FormatISO8601(time.Now())
}

func ParseISO8601(s string) (time.Time, error) {
	return time.Parse(TIMEFORMAT_ISO8601, s)
}

type TimeSync struct {
	ntpServer string
	syncCmd   string
}

func NewTimeSync(ntpServer string, syncCmd string) (*TimeSync, error) {
	ts := &TimeSync{
		ntpServer: ntpServer,
		syncCmd:   syncCmd,
	}

	if _, err := ts.Query(); err != nil {
		return nil, err
	}

	return ts, nil
}

func (ts *TimeSync) Start() {
	log.Debug("starting to sync time periodically")

	go func() {
		for _ = range time.NewTicker(time.Second * 60).C {
			go func() {
				if err := ts.Sync(); err != nil {
					log.Error("failed to sync time", "error", err)
				}
			}()
		}
	}()
}

func (ts *TimeSync) Query() (*ntp.Response, error) {
	return ntp.Query(ts.ntpServer)
}

func (ts *TimeSync) checkNTPOffset() (bool, error) {
	resp, err := ts.Query()
	if err != nil {
		return false, err
	}

	log.Debug("check time difference", "offset", resp.ClockOffset, "allowed", MaxTimeDiffAllow)
	if resp.ClockOffset > MaxTimeDiffAllow || resp.ClockOffset < (MaxTimeDiffAllow*-1) {
		return false, nil
	}

	return true, nil
}

func (ts *TimeSync) Sync() error {
	if len(ts.syncCmd) < 1 {
		return nil
	}

	log.Debug("trying to sync time")

	var b bytes.Buffer
	cmd := exec.Command("sh", "-c", ts.syncCmd)
	cmd.Stderr = &b
	cmd.Stdout = &b

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run time-sync-command: `%s`: %s", ts.syncCmd, b.String())
	} else {
		log.Debug("time-sync-command was executed", "output", b.String())
	}

	var err error
	var synced bool
	for i := 0; i < 3; i++ {
		time.Sleep(time.Second * 3)
		if synced, err = ts.checkNTPOffset(); err != nil {
			return err
		}
		if synced {
			break
		}
	}

	if !synced {
		return fmt.Errorf("time-sync-command was executed, but still not synced")
	}

	log.Debug("time synced")

	return nil
}
