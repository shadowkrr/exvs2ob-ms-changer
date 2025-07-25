// +build windows

package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"unsafe"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	"golang.org/x/sys/windows"
)

type Unit struct {
	ID    int32
	Label string
}

const (
	processName = "vsac27_Release_CLIENT.exe"
)

var (
	unitMap   = map[string]Unit{}
	stopChan  chan struct{}
)

func main() {
	a := app.New()
	w := a.NewWindow("ms-changer")
	w.Resize(fyne.NewSize(700, 1000))

	statusBind := binding.NewString()
	status := widget.NewLabelWithData(statusBind)

	var startButton *widget.Button

	// Check game process at startup
	if pid, err := getProcessID(processName); err == nil && pid > 0 {
		statusBind.Set(fmt.Sprintf("✅ Game process found: PID %d", pid))
	} else {
		statusBind.Set("🕹️ Waiting for game process...")
	}

	unitLabels := loadUnitsFromCSV("units.csv")
	if len(unitLabels) == 0 {
		statusBind.Set("❌ Failed to load units.csv")
		return
	}

	radio := widget.NewRadioGroup(unitLabels, nil)
	radio.Horizontal = false
	radio.Selected = unitLabels[0]
	scroll := container.NewScroll(radio)
	scroll.SetMinSize(fyne.NewSize(660, 900))

	startButton = widget.NewButton("Start Writing", func() {
		if stopChan != nil {
			statusBind.Set("⚠️ Already running")
			return
		}
	
		stopChan = make(chan struct{})
		selected := radio.Selected
		unit := unitMap[selected]
	
		statusBind.Set(fmt.Sprintf("🚀 Writing started for: %s (%d)", unit.Label, unit.ID))
		startButton.Disable() // 🔒 Disable button while running
	
		go func() {
			for {
				select {
				case <-stopChan:
					statusBind.Set("⏹ Writing stopped.")
					stopChan = nil
					startButton.Enable() // 🔓 Re-enable button
					return
				default:
					cmd := exec.Command("./ms-changer-gui-cli.exe", strconv.Itoa(int(unit.ID)))
					cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

					out, err := cmd.CombinedOutput()
					if err != nil {
						statusBind.Set(fmt.Sprintf("❌ CLI error: %v", err))
					} else {
						statusBind.Set(string(out))
					}
					time.Sleep(1 * time.Second)
				}
			}
		}()
	})
	
	stopButton := widget.NewButton("Stop", func() {
		if stopChan != nil {
			close(stopChan)
		}
	})

	w.SetContent(container.NewVBox(
		widget.NewLabel("Select Mobile Suit:"),
		scroll,
		startButton,
		stopButton,
		status,
	))

	w.ShowAndRun()
}

func waitForGame(name string) uint32 {
	for {
		pid, err := getProcessID(name)
		if err == nil {
			return pid
		}
		fmt.Println("🔁 Still waiting for process:", name)
		time.Sleep(2 * time.Second)
	}
}

func loadUnitsFromCSV(filename string) []string {
	f, err := os.Open(filename)
	if err != nil {
		fmt.Println("❌ Failed to open CSV:", err)
		return nil
	}
	defer f.Close()

	records, err := csv.NewReader(f).ReadAll()
	if err != nil {
		fmt.Println("❌ Failed to parse CSV:", err)
		return nil
	}

	var labels []string
	for i, r := range records {
		if i == 0 || len(r) < 3 {
			continue
		}
		id, _ := strconv.Atoi(r[0])
		unitID, _ := strconv.Atoi(r[2])
		label := fmt.Sprintf("%d: %s", id, r[1])

		unitMap[label] = Unit{
			ID:    int32(unitID),
			Label: r[1],
		}
		labels = append(labels, label)
	}
	return labels
}

func getProcessID(name string) (uint32, error) {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return 0, err
	}
	defer windows.CloseHandle(snap)

	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	err = windows.Process32First(snap, &entry)
	for err == nil {
		exe := syscall.UTF16ToString(entry.ExeFile[:])
		if exe == name {
			return entry.ProcessID, nil
		}
		err = windows.Process32Next(snap, &entry)
	}
	return 0, fmt.Errorf("process %s not found", name)
}
