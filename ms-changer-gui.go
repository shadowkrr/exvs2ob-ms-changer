
// +build windows

package main

import (
    "encoding/csv"
    "fmt"
    "os"
    "os/exec"
    "strconv"
    "strings"
    "syscall"
    "unsafe"
    "time"
	"sort"

    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/app"
    "fyne.io/fyne/v2/container"
    "fyne.io/fyne/v2/data/binding"
    "fyne.io/fyne/v2/widget"
    "golang.org/x/sys/windows"
)

type Unit struct {
    ID    int32
    Title string
    MS    string
    Value int32
}

const processName = "vsac27_Release_CLIENT.exe"

var (
    allUnits []Unit
    stopChan chan struct{}
)

func main() {
    a := app.New()
    w := a.NewWindow("ms-changer")
    w.Resize(fyne.NewSize(700, 1000))

    statusBind := binding.NewString()
    status := widget.NewLabelWithData(statusBind)

    var startButton *widget.Button
    selectedID := binding.NewString()

    // Check if game is running
    if pid, err := getProcessID(processName); err == nil && pid > 0 {
        statusBind.Set(fmt.Sprintf("✅ Game process found: PID %d", pid))
    } else {
        statusBind.Set("🕹️ Waiting for game process...")
    }

    allUnits = loadUnitsFromCSV("units.csv")
    if len(allUnits) == 0 {
        statusBind.Set("❌ Failed to load units.csv")
        return
    }

	var radioItems []string
	var unitMap = map[string]Unit{}

	for _, unit := range allUnits {
		label := fmt.Sprintf("[%s] %s", unit.Title, unit.MS)
		radioItems = append(radioItems, label)
		unitMap[label] = unit
	}

	radio := widget.NewRadioGroup(radioItems, func(selected string) {
		unit := unitMap[selected]
		selectedID.Set(strconv.Itoa(int(unit.Value)))
	})
	radio.Horizontal = false
	radio.Selected = radioItems[0]

	scroll := container.NewVScroll(radio)
	scroll.SetMinSize(fyne.NewSize(660, 900))


    startButton = widget.NewButton("Start Writing", func() {
        if stopChan != nil {
            statusBind.Set("⚠️ Already running")
            return
        }

        unitValueStr, err := selectedID.Get()
        if err != nil || unitValueStr == "" {
            statusBind.Set("❌ No unit selected")
            return
        }

        stopChan = make(chan struct{})
        unitValue := unitValueStr
        statusBind.Set(fmt.Sprintf("🚀 Writing started for ID: %s", unitValue))
        startButton.Disable()

        go func() {
            for {
                select {
                case <-stopChan:
                    statusBind.Set("⏹ Writing stopped.")
                    stopChan = nil
                    startButton.Enable()
                    return
                default:
                    cmd := exec.Command("./ms-changer-gui-cli.exe", unitValue)
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

func loadUnitsFromCSV(filename string) []Unit {
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

	var units []Unit
	titleOrder := map[string]int{}
	orderCounter := 0

	for i, r := range records {
		if i == 0 || len(r) < 4 {
			continue
		}
		id, _ := strconv.Atoi(r[0])
		value, _ := strconv.Atoi(r[3])
		title := r[1]

		// 作品の出現順に記録
		if _, exists := titleOrder[title]; !exists {
			titleOrder[title] = orderCounter
			orderCounter++
		}

		units = append(units, Unit{
			ID:    int32(id),
			Title: title,
			MS:    r[2],
			Value: int32(value),
		})
	}

	// 🔽 出現順 → ID昇順 でソート
	sort.Slice(units, func(i, j int) bool {
		ti := titleOrder[units[i].Title]
		tj := titleOrder[units[j].Title]
		if ti == tj {
			return units[i].ID < units[j].ID
		}
		return ti < tj
	})

	return units
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
        if strings.EqualFold(exe, name) {
            return entry.ProcessID, nil
        }
        err = windows.Process32Next(snap, &entry)
    }
    return 0, fmt.Errorf("process %s not found", name)
}
