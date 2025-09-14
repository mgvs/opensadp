package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"

	"opensadp/internal/sadp"
)

var deviceFields = []string{
	"Uuid", "Types", "DeviceType", "DeviceDescription", "DeviceSN",
	"CommandPort", "HttpPort", "MAC", "IPv4Address", "IPv4SubnetMask",
	"IPv4Gateway", "IPv6Address", "IPv6Gateway", "IPv6MaskLen", "DHCP",
	"AnalogChannelNum", "DigitalChannelNum", "SoftwareVersion", "DSPVersion",
	"BootTime", "Encrypt", "ResetAbility", "DiskNumber", "Activated",
	"PasswordResetAbility", "PasswordResetModeSecond", "DetailOEMCode",
	"SupportSecurityQuestion", "SupportHCPlatform", "HCPlatformEnable",
	"IsModifyVerificationCode", "Salt", "DeviceLock", "SDKServerStatus",
	"SDKOverTLSServerStatus", "SDKOverTLSPort", "SupportMailBox", "supportEzvizUnbind",
}

func main() {
	c, err := sadp.NewClient(37020, 2*time.Second)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	defer c.Close()

	probe := sadp.Message{
		Uuid:  uuid.New().String(),
		MAC:   "ff-ff-ff-ff-ff-ff",
		Types: "inquiry",
	}
	if _, err := c.WriteMessage(probe); err != nil {
		fmt.Fprintln(os.Stderr, "send error:", err)
		os.Exit(1)
	}

	var rows []map[string]string
	for {
		buf, _, err := c.ReceiveOnce()
		if err != nil {
			break // deadline or no more responses
		}
		m, err := sadp.UnmarshalResponse(buf)
		if err != nil || m == nil {
			continue
		}
		rows = append(rows, m)
	}

	w := csv.NewWriter(os.Stdout)
	_ = w.Write(deviceFields)
	for _, d := range rows {
		rec := make([]string, len(deviceFields))
		for i, k := range deviceFields {
			rec[i] = d[k]
		}
		_ = w.Write(rec)
	}
	w.Flush()
}
