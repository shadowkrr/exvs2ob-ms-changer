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
	mainTabs *container.AppTabs
	progressBar *widget.ProgressBarInfinite
	radioGroups map[string]*widget.RadioGroup
	currentTabIndex int
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

	// Initialize radio groups map
	radioGroups = make(map[string]*widget.RadioGroup)
	
	// Create accordion with grouped units
	accordion = container.NewAppTabs()
	accordion.SetTabLocation(container.TabLocationTop)
	updateAccordion("", selectedID)
	
	// Add tab selection listener
	accordion.OnChanged = func(tab *container.TabItem) {
		// Update current tab index for keyboard navigation
		for i, item := range accordion.Items {
			if item == tab {
				currentTabIndex = i
				break
			}
		}
		
		// Ensure selection is maintained when switching tabs
		if tab != nil && radioGroups[tab.Text] != nil {
			radio := radioGroups[tab.Text]
			if radio.Selected != "" {
				// Keep existing selection
			} else if len(radio.Options) > 0 {
				// Auto-select first item if nothing selected
				radio.SetSelected(radio.Options[0])
			}
		}
	}

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

	// Create Mobile Suit selection page
	selectorHeader := container.NewVBox(
		widget.NewRichTextFromMarkdown("## 🤖 Mobile Suit Selection"),
		searchEntry,
		widget.NewSeparator(),
	)

	selectorContent := container.NewVScroll(accordion)
	selectorContent.SetMinSize(fyne.NewSize(850, 350))

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

	selectorFooter := container.NewVBox(
		widget.NewSeparator(),
		buttonContainer,
		statusContainer,
	)

	selectorPage := container.NewBorder(
		selectorHeader, selectorFooter, nil, nil,
		selectorContent,
	)

	// Create main tab container
	mainTabs = container.NewAppTabs()
	mainTabs.SetTabLocation(container.TabLocationTop)
	
	// Add Mobile Suit selection tab
	mainTabs.Append(container.NewTabItem("🤖 Mobile Suits", selectorPage))
	
	// Add other pages
	createAdditionalPages()

	w.SetContent(mainTabs)

	w.ShowAndRun()
}

func createAdditionalPages() {
	// About page
	aboutContent := widget.NewRichTextFromMarkdown(`# 📋 About MS Changer

**MS Changer** is a utility for modifying the in-game Mobile Suit selection of a Windows-based arcade client via memory editing.

## ✨ Features
- 🎮 **Real-time Mobile Suit switching** during gameplay
- 🔍 **Search functionality** to quickly find your favorite Mobile Suit
- 📁 **Organized by series** with intuitive tab navigation
- 🚀 **Easy-to-use GUI** with visual feedback
- ⚡ **Instant application** of changes

## 🎯 Supported Games
- **EXVS2 Overboost** (vsac27_Release_CLIENT.exe)

## ⚠️ Important Notes
- ✅ **Run as Administrator** for memory access
- 🛡️ **For educational and personal use only**
- 🕹️ **Game must be running** before starting operations

---
*Version 2.0 - Enhanced GUI Edition*`)

	aboutScroll := container.NewVScroll(aboutContent)
	aboutScroll.SetMinSize(fyne.NewSize(850, 500))
	
	// Usage Instructions page
	usageContent := widget.NewRichTextFromMarkdown(`# 📖 Usage Instructions

## 🚀 Getting Started

### 1. Prerequisites
- Windows operating system
- Administrator privileges
- Target game process running

### 2. Launch Sequence
1. **Start the game** (vsac27_Release_CLIENT.exe)
2. **Run MS Changer as Administrator**
3. Wait for the green "✅ Game process found" message

### 3. Select Mobile Suit
1. Use the **🔍 Search** box to filter Mobile Suits
2. **Click on tabs** to browse by series (🚀 Gundam, ⭐ MSV, etc.)
3. **Select your desired Mobile Suit** from the radio buttons

### 4. Apply Changes
1. Click **🚀 Start Writing** to begin memory modification
2. The progress bar will show **⏳ Writing...** status  
3. **Switch to game** and observe the Mobile Suit change
4. Click **⏹ Stop** when finished

## 💡 Tips
- Search works for both Mobile Suit names and series titles
- Tab numbers show how many Mobile Suits are in each series
- Status messages provide real-time feedback
- Changes apply immediately while writing is active

## 🔧 Troubleshooting
- Ensure game is running before starting
- Run as Administrator if process access fails
- Check that the correct .exe and .csv files are present`)

	usageScroll := container.NewVScroll(usageContent)
	usageScroll.SetMinSize(fyne.NewSize(850, 500))

	// Settings/Info page
	settingsContent := widget.NewRichTextFromMarkdown(`# ⚙️ Settings & Information

## 📁 File Structure
- **ms-changer-gui.exe** - Main GUI application
- **ms-changer-gui-cli.exe** - CLI helper (called by GUI)
- **ms-changer.exe** - Standalone CLI version
- **units.csv** - Mobile Suit database

## 🔧 Memory Configuration
- **Base Pointer Offset**: 0x1EA0708
- **Fixed Offset**: 0x524
- **Target Process**: vsac27_Release_CLIENT.exe
- **Write Interval**: 1 second

## 📊 CSV Format
` + "```" + `csv
id,title,ms,value
1,機動戦士ガンダム,ガンダム,1001001
2,機動戦士ガンダム,シャア専用ゲルググ,1002001
` + "```" + `

## 🎨 Interface Elements
- **Tabs**: Series organization with count display
- **Search**: Real-time filtering of Mobile Suits
- **Progress**: Visual feedback during operations
- **Status**: Detailed operation information

## ⚠️ Safety Notes
- Always backup save data before use
- Only use with legitimate game copies
- Respect online play guidelines
- Educational purposes only`)

	settingsScroll := container.NewVScroll(settingsContent)
	settingsScroll.SetMinSize(fyne.NewSize(850, 500))

	// Add all pages to main tabs
	mainTabs.Append(container.NewTabItem("📋 About", aboutScroll))
	mainTabs.Append(container.NewTabItem("📖 Usage", usageScroll))
	mainTabs.Append(container.NewTabItem("⚙️ Settings", settingsScroll))
}

func updateAccordion(searchQuery string, selectedID binding.String) {
	// Clear existing tabs and radio groups
	for len(accordion.Items) > 0 {
		accordion.RemoveIndex(0)
	}
	radioGroups = make(map[string]*widget.RadioGroup)
	
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
			
			// Add emoji based on series
			titleIcon := "📺"
			if strings.Contains(title, "ガンダム") {
				titleIcon = "🚀"
			} else if strings.Contains(title, "MSV") {
				titleIcon = "⭐"
			}
			
			tabTitle := fmt.Sprintf("%s %s (%d)", titleIcon, title, len(units))
			
			// Store radio group for tab switching
			radioGroups[tabTitle] = radio
			
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
			
			tabItem := container.NewTabItem(tabTitle, scrollContent)
			accordion.Append(tabItem)
		}
	}
	
	// If no results found, show message
	if len(accordion.Items) == 0 {
		noResultsLabel := widget.NewLabel("🔍 No Mobile Suits found matching your search")
		noResultsLabel.Alignment = fyne.TextAlignCenter
		accordion.Append(container.NewTabItem("❌ No Results", noResultsLabel))
	} else {
		// Auto-select the first tab
		if len(accordion.Items) > 0 {
			accordion.SelectTab(accordion.Items[0])
		}
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
