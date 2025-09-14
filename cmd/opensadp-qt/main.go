package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mappu/miqt/qt"

	"opensadp/internal/sadp"
	_ "opensadp/res"
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

// Columns in the table
var tableHeaders = []string{
	"ID",
	"Device Type",
	"Status",
	"IPv4 Address",
	"Port",
	"Enhanced SDK Service Port",
	"Software Version",
	"IPv4 Gateway",
	"HTTP",
	"Device Serial No.",
	"Subnet Mask",
	"MAC Address",
	"Encoding Channel(s)",
	"DSP Version",
	"Start Time",
	"IPv6 Address",
	"IPv6 GateWay",
	"IPv6 Prefix Length",
	"Support IPv6",
	"IPv6 Modifiable",
	"Support DHCP",
	"IPv4 DHCP Status",
	"Support Hik-Connect",
	"Hik-Connect Status",
}

type appState struct {
	win        *qt.QMainWindow
	countLabel *qt.QLabel
	exportBtn  *qt.QPushButton
	filterEdit *qt.QLineEdit
	table      *qt.QTableWidget

	allRows      []map[string]string
	filteredRows []map[string]string

	resultsCh chan []map[string]string
}

func main() {
	qt.NewQApplication(os.Args)
	qt.QCoreApplication_SetApplicationName("OpenSADP")
	qt.QGuiApplication_SetApplicationDisplayName("OpenSADP")
	// Set application icon from embedded resource
	qt.QApplication_SetWindowIcon(qt.NewQIcon4(":/opensadp/sadp.ico"))
	state := &appState{resultsCh: make(chan []map[string]string, 1)}
	state.buildUI()
	state.win.Show()

	// Timer to poll for scan results coming from goroutine
	timer := qt.NewQTimer()
	timer.OnTimeout(func() {
		select {
		case rows := <-state.resultsCh:
			state.allRows = rows
			state.applyFilter()
			state.updateTopBar()
		default:
		}
	})
	timer.Start(200)

	// Kick off initial scan immediately
	state.startScan()

	os.Exit(qt.QApplication_Exec())
}

func (s *appState) buildUI() {
	s.win = qt.NewQMainWindow(nil)
	s.win.SetWindowTitle("OpenSADP - Device Discovery")
	s.win.Resize(1100, 600)

	central := qt.NewQWidget(nil)
	vbox := qt.NewQVBoxLayout(central)
	vbox.SetContentsMargins(0, 0, 0, 0)
	vbox.SetSpacing(0)

	// Top bar
	bar := qt.NewQWidget(nil)
	bar.SetObjectName("TopBar")
	h := qt.NewQHBoxLayout(bar)
	h.SetContentsMargins(16, 12, 16, 12)
	h.SetSpacing(12)

	s.countLabel = qt.NewQLabel3("Total number of online devices: <span style='color:#1e90ff; font-weight:600;'>0</span>")
	s.countLabel.SetTextFormat(qt.RichText)
	h.AddWidget(s.countLabel.QWidget)

	unbind := qt.NewQPushButton3("Unbind")
	unbind.SetEnabled(false)
	unbind.SetProperty("variant", qt.NewQVariant14("danger"))
	h.AddWidget(unbind.QWidget)

	s.exportBtn = qt.NewQPushButton3("Export")
	s.exportBtn.SetEnabled(false)
	s.exportBtn.SetProperty("variant", qt.NewQVariant14("danger"))
	s.exportBtn.OnPressed(func() { s.exportCsv() })
	h.AddWidget(s.exportBtn.QWidget)

	refresh := qt.NewQPushButton3("Refresh")
	refresh.SetProperty("variant", qt.NewQVariant14("primary"))
	refresh.OnPressed(func() { s.startScan() })
	h.AddWidget(refresh.QWidget)

	filterWrap := qt.NewQWidget(nil)
	fl := qt.NewQHBoxLayout(filterWrap)
	fl.SetContentsMargins(0, 0, 0, 0)
	fl.SetSpacing(6)
	s.filterEdit = qt.NewQLineEdit(nil)
	s.filterEdit.SetPlaceholderText("Filter")
	s.filterEdit.OnTextChanged(func(string) { s.applyFilter() })
	fl.AddWidget(s.filterEdit.QWidget)
	srch := qt.NewQToolButton(nil)
	srch.SetText("🔍")
	srch.OnPressed(func() { s.applyFilter() })
	fl.AddWidget(srch.QWidget)
	filterWrap.SetMaximumWidth(320)
	h.AddWidget(filterWrap)

	// Write a temporary SVG checkmark and reference it in the stylesheet so the checkmark is visible cross-platform
	checkSVG := `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16"><polyline points="3,9 7,13 13,5" fill="none" stroke="#ffffff" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/></svg>`
	checkPath := filepath.Join(os.TempDir(), "opensadp_check.svg")
	_ = os.WriteFile(checkPath, []byte(checkSVG), 0o644)
	style := fmt.Sprintf(`#TopBar{background:#f7f7f7; border-bottom:1px solid #e1e1e1;}
QPushButton{padding:8px 16px; border-radius:6px;}
QPushButton[variant="primary"]{background:#f0f0f0; border:1px solid #d0d0d0;}
QPushButton[variant="primary"]:hover{background:#e8e8e8;}
QPushButton[variant="primary"]:pressed{background:#dcdcdc;}
QPushButton[variant="danger"]{background:#f7d6d6; border:1px solid #e6bcbc; color:#7d3b3b;}
QPushButton[variant="danger"]:hover{background:#f2c0c0;}
QPushButton[variant="danger"]:pressed{background:#e9aaaa;}
QLineEdit{padding:8px 12px; border:1px solid #d0d0d0; border-radius:6px;}
QCheckBox::indicator{width:16px; height:16px; border-radius:4px; border:1px solid #d0d0d0; background:#ffffff;}
QCheckBox::indicator:hover{background:#f7d6d6; border:1px solid #e6bcbc;}
QCheckBox::indicator:checked{background:#f7d6d6; border:1px solid #e6bcbc; image: url('%s');}
QCheckBox::indicator:checked:hover{background:#f2c0c0; border:1px solid #e6bcbc; image: url('%s');}
`, checkPath, checkPath)
	s.win.SetStyleSheet(style)

	vbox.AddWidget(bar)

	// Table
	s.table = qt.NewQTableWidget(nil)
	s.table.SetColumnCount(len(tableHeaders))
	for i, name := range tableHeaders {
		item := qt.NewQTableWidgetItem2(name)
		s.table.SetHorizontalHeaderItem(i, item)
	}
	s.table.SetAlternatingRowColors(true)
	s.table.SetSelectionBehavior(qt.QAbstractItemView__SelectRows)
	s.table.SetSelectionMode(qt.QAbstractItemView__SingleSelection)
	s.table.SetSortingEnabled(true)
	// Header config to fill until dock and allow reordering/resizing
	header := s.table.HorizontalHeader()
	header.SetStretchLastSection(true)
	header.SetSectionsMovable(true)
	header.SetSectionResizeMode(qt.QHeaderView__Interactive)

	vbox.AddWidget2(s.table.QWidget, 1)
	central.SetLayout(vbox.QLayout)
	s.win.SetCentralWidget(central)

	// Details dock setup
	dock := qt.NewQDockWidget2("Modify Network Parameters")
	dock.SetFeatures(qt.QDockWidget__DockWidgetClosable)
	panel := qt.NewQWidget(nil)
	form := qt.NewQFormLayout(panel)
	form.SetFieldGrowthPolicy(qt.QFormLayout__ExpandingFieldsGrow)
	form.SetLabelAlignment(qt.AlignRight)
	mk := func() *qt.QLineEdit {
		e := qt.NewQLineEdit(nil)
		e.SetReadOnly(true)
		e.SetMinimumWidth(0)
		// Make the value field expand horizontally with the dock
		e.SetSizePolicy2(qt.QSizePolicy__Expanding, qt.QSizePolicy__Fixed)
		return e
	}
	dhcp := qt.NewQCheckBox(nil)
	hc := qt.NewQCheckBox(nil)
	tSN, tIp, tPort, tSdk, tMask, tGw := mk(), mk(), mk(), mk(), mk(), mk()
	tIpv6, tIpv6Gw, tIpv6Pref, tHttp := mk(), mk(), mk(), mk()
	form.AddRow3("Enable DHCP:", dhcp.QWidget)
	form.AddRow3("Enable Hik-Connect:", hc.QWidget)
	form.AddRow3("Device Serial No.:", tSN.QWidget)
	form.AddRow3("IP Address:", tIp.QWidget)
	form.AddRow3("Port:", tPort.QWidget)
	form.AddRow3("Enhanced SDK Service Port:", tSdk.QWidget)
	form.AddRow3("Subnet Mask:", tMask.QWidget)
	form.AddRow3("Gateway:", tGw.QWidget)
	form.AddRow3("IPv6 Address:", tIpv6.QWidget)
	form.AddRow3("IPv6 Gateway:", tIpv6Gw.QWidget)
	form.AddRow3("IPv6 Prefix Length:", tIpv6Pref.QWidget)
	form.AddRow3("HTTP Port:", tHttp.QWidget)
	scroll := qt.NewQScrollArea(nil)
	scroll.SetWidgetResizable(true)
	scroll.SetWidget(panel)
	dock.SetWidget(scroll.QWidget)
	s.win.AddDockWidget(qt.RightDockWidgetArea, dock)
	dock.Hide()

	// Selection handler to populate details and show dock
	s.table.OnItemSelectionChanged(func() {
		r := s.table.CurrentRow()
		if r < 0 {
			dock.Hide()
			return
		}
		get := func(col int) string {
			it := s.table.Item(r, col)
			if it == nil {
				return ""
			}
			return it.Text()
		}
		dhcp.SetChecked(false)
		hc.SetChecked(false)
		tSN.SetText(get(9))
		tIp.SetText(get(3))
		tPort.SetText(get(4))
		tSdk.SetText(get(5))
		tMask.SetText(get(10))
		tGw.SetText(get(7))
		tHttp.SetText(get(8))
		dock.Show()
	})

	// DHCP toggle behavior: when enabled, make fields read-only and gray
	dhcp.OnStateChanged(func(_ int) {
		isOn := dhcp.IsChecked()
		set := func(le *qt.QLineEdit, ro bool) {
			le.SetReadOnly(ro)
			if ro {
				le.SetStyleSheet("QLineEdit{background:#f5f5f5; color:#808080}")
			} else {
				le.SetStyleSheet("")
			}
		}
		set(tIp, isOn)
		// Keep ports editable even when DHCP is enabled
		set(tPort, false)
		set(tSdk, false)
		set(tMask, isOn)
		set(tGw, isOn)
		set(tIpv6, isOn)
		set(tIpv6Gw, isOn)
		set(tIpv6Pref, isOn)
		set(tHttp, isOn)
	})

	// Menu bar with About action
	mb := s.win.MenuBar()
	if mb == nil {
		mb = qt.NewQMenuBar(nil)
		s.win.SetMenuBar(mb)
	}
	help := mb.AddMenuWithTitle("Help")
	aboutAct := help.AddAction("About OpenSADP")
	aboutAct.OnTriggered(func() {
		ver := "(unknown)"
		f := qt.NewQFile2(":/opensadp/version.txt")
		if f.Open(qt.QIODevice__ReadOnly) {
			data := f.ReadAll()
			f.Close()
			if len(data) > 0 {
				ver = strings.TrimSpace(string(data))
			}
		}
		qt.QMessageBox_Information(s.win.QWidget, "About OpenSADP", "OpenSADP version "+ver)
	})
}

func (s *appState) updateTopBar() {
	n := len(s.allRows)
	s.countLabel.SetText("Total number of online devices: <span style='color:#1e90ff; font-weight:600;'>" + itoa(n) + "</span>")
	s.exportBtn.SetEnabled(n > 0)
}

func (s *appState) startScan() {
	go func(ch chan []map[string]string) {
		client, err := sadp.NewClient(37020, 2*time.Second)
		if err != nil {
			ch <- nil
			return
		}
		defer client.Close()
		probe := sadp.Message{Uuid: strings.ToUpper(uuid.New().String()), MAC: "ff-ff-ff-ff-ff-ff", Types: "inquiry"}
		_, _ = client.WriteMessage(probe)
		var rows []map[string]string
		for {
			buf, _, err := client.ReceiveOnce()
			if err != nil {
				break
			}
			m, err := sadp.UnmarshalResponse(buf)
			if err != nil || m == nil {
				continue
			}
			rows = append(rows, m)
		}
		ch <- rows
	}(s.resultsCh)
}

func (s *appState) applyFilter() {
	query := strings.TrimSpace(s.filterEdit.Text())
	query = strings.ToLower(query)
	if query == "" {
		s.filteredRows = s.allRows
		s.refreshTable()
		return
	}
	var filtered []map[string]string
	for _, d := range s.allRows {
		hay := strings.ToLower(strings.Join([]string{
			d["IPv4Address"],
			d["DeviceSN"],
			d["DeviceDescription"],
			d["DeviceType"],
		}, " "))
		if strings.Contains(hay, query) {
			filtered = append(filtered, d)
		}
	}
	s.filteredRows = filtered
	s.refreshTable()
}

func (s *appState) refreshTable() {
	rows := s.filteredRows
	s.table.SetRowCount(len(rows))
	for i, d := range rows {
		// ID
		s.table.SetItem(i, 0, qt.NewQTableWidgetItem2(pad3(i+1)))
		// Device Type
		typeVal := d["DeviceDescription"]
		if typeVal == "" {
			typeVal = d["DeviceType"]
		}
		s.table.SetItem(i, 1, qt.NewQTableWidgetItem2(typeVal))
		// Status
		s.table.SetItem(i, 2, qt.NewQTableWidgetItem2("Active"))
		// IPv4 Address
		s.table.SetItem(i, 3, qt.NewQTableWidgetItem2(d["IPv4Address"]))
		// Port (CommandPort)
		s.table.SetItem(i, 4, qt.NewQTableWidgetItem2(d["CommandPort"]))
		// Enhanced SDK Service Port
		sdk := d["SDKOverTLSPort"]
		if sdk == "" {
			sdk = d["EnhancedSDKServicePort"]
		}
		s.table.SetItem(i, 5, qt.NewQTableWidgetItem2(sdk))
		// Software Version
		s.table.SetItem(i, 6, qt.NewQTableWidgetItem2(d["SoftwareVersion"]))
		// IPv4 Gateway
		s.table.SetItem(i, 7, qt.NewQTableWidgetItem2(d["IPv4Gateway"]))
		// HTTP
		s.table.SetItem(i, 8, qt.NewQTableWidgetItem2(d["HttpPort"]))
		// Device SN
		s.table.SetItem(i, 9, qt.NewQTableWidgetItem2(d["DeviceSN"]))
		// Subnet Mask
		s.table.SetItem(i, 10, qt.NewQTableWidgetItem2(d["IPv4SubnetMask"]))
		// MAC Address
		s.table.SetItem(i, 11, qt.NewQTableWidgetItem2(d["MAC"]))
		// Encoding Channel(s)
		s.table.SetItem(i, 12, qt.NewQTableWidgetItem2(d["DigitalChannelNum"]))
		// DSP Version
		s.table.SetItem(i, 13, qt.NewQTableWidgetItem2(d["DSPVersion"]))
		// Start Time
		s.table.SetItem(i, 14, qt.NewQTableWidgetItem2(d["BootTime"]))
		// IPv6 Address
		s.table.SetItem(i, 15, qt.NewQTableWidgetItem2(d["IPv6Address"]))
		// IPv6 GateWay
		s.table.SetItem(i, 16, qt.NewQTableWidgetItem2(d["IPv6Gateway"]))
		// IPv6 Prefix Length
		s.table.SetItem(i, 17, qt.NewQTableWidgetItem2(d["IPv6MaskLen"]))
		// Support IPv6 (best-effort)
		supIPv6 := "No"
		if v := d["IPv6Address"]; v != "" && v != "::" {
			supIPv6 = "Yes"
		}
		s.table.SetItem(i, 18, qt.NewQTableWidgetItem2(supIPv6))
		// IPv6 Modifiable (unknown -> N/A)
		s.table.SetItem(i, 19, qt.NewQTableWidgetItem2("N/A"))
		// Support DHCP
		s.table.SetItem(i, 20, qt.NewQTableWidgetItem2("Yes"))
		// IPv4 DHCP Status
		dhcpStatus := strings.ToLower(d["DHCP"])
		dhcpLabel := "OFF"
		if dhcpStatus == "true" || dhcpStatus == "1" || dhcpStatus == "yes" {
			dhcpLabel = "ON"
		}
		s.table.SetItem(i, 21, qt.NewQTableWidgetItem2(dhcpLabel))
		// Support Hik-Connect
		supHC := "No"
		if v := strings.ToLower(d["SupportHCPlatform"]); v == "true" || v == "1" || v == "yes" {
			supHC = "Yes"
		}
		s.table.SetItem(i, 22, qt.NewQTableWidgetItem2(supHC))
		// Hik-Connect Status
		hcStatus := strings.ToLower(d["HCPlatformEnable"])
		hcLabel := "OFF"
		if hcStatus == "true" || hcStatus == "1" || hcStatus == "yes" {
			hcLabel = "ON"
		}
		s.table.SetItem(i, 23, qt.NewQTableWidgetItem2(hcLabel))
	}
	s.table.ResizeColumnsToContents()
}

func (s *appState) exportCsv() {
	filename := qt.QFileDialog_GetSaveFileName()
	if filename == "" {
		return
	}
	path := filename
	f, err := os.Create(path)
	if err != nil {
		return
	}
	defer f.Close()
	w := csv.NewWriter(f)
	_ = w.Write(deviceFields)
	for _, d := range s.filteredRows {
		rec := make([]string, len(deviceFields))
		for i, k := range deviceFields {
			rec[i] = d[k]
		}
		_ = w.Write(rec)
	}
	w.Flush()
}

func pad3(n int) string {
	if n < 10 {
		return "00" + itoa(n)
	}
	if n < 100 {
		return "0" + itoa(n)
	}
	return itoa(n)
}

func itoa(i int) string { return strconv.Itoa(i) }
