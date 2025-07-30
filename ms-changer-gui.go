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
	"fyne.io/fyne/v2/theme"
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
	selectedUnit *Unit
	searchEntry *widget.Entry
	accordion *container.AppTabs
	progressBar *widget.ProgressBarInfinite
)

func main() {
	a := app.New()
	a.SetIcon(theme.ComputerIcon())
	w := a.NewWindow("🤖 MS Changer - Mobile Suit Selector")
	w.Resize(fyne.NewSize(900, 700))
	w.CenterOnScreen()

	statusBind := binding.NewString()
	status := widget.NewLabelWithData(statusBind)
	status.Wrapping = fyne.TextWrapWord
	
	// Create progress bar (initially hidden)
	progressBar = widget.NewProgressBarInfinite()
	progressBar.Hide()

	var startButton *widget.Button
	selectedID := binding.NewString()

	// Check if game process is running
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

	// Create search functionality
	searchEntry = widget.NewEntry()
	searchEntry.SetPlaceHolder("🔍 Search Mobile Suit...")
	searchEntry.OnChanged = func(query string) {
		updateAccordion(query, selectedID)
	}

	// Create accordion with grouped units
	accordion = container.NewAppTabs()
	updateAccordion("", selectedID)

	// Set default selection
	if len(allUnits) > 0 {
		selectedUnit = &allUnits[0]
		selectedID.Set(strconv.Itoa(int(selectedUnit.Value)))
	}

	startButton = widget.NewButton("🚀 Start Writing", func() {
		if stopChan != nil {
			statusBind.Set("⚠️ Already running")
			return
		}

		unitValueStr, err := selectedID.Get()
		if err != nil || unitValueStr == "" || selectedUnit == nil {
			statusBind.Set("❌ No Mobile Suit selected")
			return
		}

		stopChan = make(chan struct{})
		unitValue := unitValueStr
		statusBind.Set(fmt.Sprintf("🚀 Writing started: %s - %s (ID: %s)", selectedUnit.Title, selectedUnit.MS, unitValue))
		startButton.Disable()
		startButton.SetText("⏳ Writing...")
		progressBar.Show()
		progressBar.Start()

		go func() {
			for {
				select {
				case <-stopChan:
					statusBind.Set("⏹ Writing stopped.")
					stopChan = nil
					startButton.Enable()
					startButton.SetText("🚀 Start Writing")
					progressBar.Stop()
					progressBar.Hide()
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
	startButton.Importance = widget.HighImportance

	stopButton := widget.NewButton("⏹ Stop", func() {
		if stopChan != nil {
			close(stopChan)
		}
	})
	stopButton.Importance = widget.MediumImportance

	// Create main layout
	header := container.NewVBox(
		widget.NewRichTextFromMarkdown("# 🤖 Mobile Suit Changer\n**Select your Mobile Suit from the list below:**"),
		searchEntry,
		widget.NewSeparator(),
	)

	mainContent := container.NewVScroll(accordion)
	mainContent.SetMinSize(fyne.NewSize(850, 400))

	buttonContainer := container.NewGridWithColumns(2,
		startButton,
		stopButton,
	)

	statusContainer := container.NewVBox(
		container.NewBorder(
			nil, nil, 
			widget.NewIcon(theme.InfoIcon()), nil,
			status,
		),
		progressBar,
	)

	footer := container.NewVBox(
		widget.NewSeparator(),
		buttonContainer,
		statusContainer,
	)

	w.SetContent(container.NewBorder(
		header, footer, nil, nil,
		mainContent,
	))

	w.ShowAndRun()
}

func updateAccordion(searchQuery string, selectedID binding.String) {
	// Clear existing tabs
	for len(accordion.Items) > 0 {
		accordion.RemoveIndex(0)
	}
	
	// Group units by title
	titleGroups := make(map[string][]Unit)
	for _, unit := range allUnits {
		// Filter by search query if provided
		if searchQuery != "" {
			if !strings.Contains(strings.ToLower(unit.MS), strings.ToLower(searchQuery)) &&
			   !strings.Contains(strings.ToLower(unit.Title), strings.ToLower(searchQuery)) {
				continue
			}
		}
		titleGroups[unit.Title] = append(titleGroups[unit.Title], unit)
	}
	
	// Sort titles
	var titles []string
	for title := range titleGroups {
		titles = append(titles, title)
	}
	sort.Strings(titles)
	
	// Create tabs for each title
	for _, title := range titles {
		units := titleGroups[title]
		
		// Create radio group for this title
		var radioItems []string
		unitMap := make(map[string]Unit)
		
		for _, unit := range units {
			label := fmt.Sprintf("🤖 %s", unit.MS)
			radioItems = append(radioItems, label)
			unitMap[label] = unit
		}
		
		if len(radioItems) > 0 {
			radio := widget.NewRadioGroup(radioItems, func(selected string) {
				if unit, exists := unitMap[selected]; exists {
					selectedUnit = &unit
					selectedID.Set(strconv.Itoa(int(unit.Value)))
				}
			})
			radio.Horizontal = false
			
			// Set default selection for first tab
			if len(accordion.Items) == 0 && len(radioItems) > 0 {
				radio.Selected = radioItems[0]
				if unit, exists := unitMap[radioItems[0]]; exists {
					selectedUnit = &unit
					selectedID.Set(strconv.Itoa(int(unit.Value)))
				}
			}
			
			scrollContent := container.NewVScroll(radio)
			scrollContent.SetMinSize(fyne.NewSize(800, 300))
			
			// Add emoji based on series
			titleIcon := "📺"
			if strings.Contains(title, "ガンダム") {
				titleIcon = "🚀"
			} else if strings.Contains(title, "MSV") {
				titleIcon = "⭐"
			}
			
			tabTitle := fmt.Sprintf("%s %s (%d)", titleIcon, title, len(units))
			accordion.Append(container.NewTabItem(tabTitle, scrollContent))
		}
	}
	
	// If no results found, show message
	if len(accordion.Items) == 0 {
		noResultsLabel := widget.NewLabel("🔍 No Mobile Suits found matching your search")
		noResultsLabel.Alignment = fyne.TextAlignCenter
		accordion.Append(container.NewTabItem("❌ No Results", noResultsLabel))
	}
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

		// Track appearance order of titles
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

	// Sort by appearance order, then by ID
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
	return 0, fmt.Errorf("Process %s not found", name)
}
