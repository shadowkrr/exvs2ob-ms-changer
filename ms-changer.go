package main

import (
    "bufio"
    "encoding/csv"
    "fmt"
    "os"
    "sort"
    "strconv"
    "strings"
    "syscall"
    "time"
    "unsafe"

    "golang.org/x/sys/windows"
)

type Unit struct {
    Name string
    ID   int32
}

const (
    PROCESS_VM_READ           = 0x0010
    PROCESS_VM_WRITE          = 0x0020
    PROCESS_VM_OPERATION      = 0x0008
    PROCESS_QUERY_INFORMATION = 0x0400
    basePointerOffset uintptr = 0x1EA0708
    fixedOffset       uintptr = 0x524
)

var unitList = make(map[int32]Unit)
var sortedIDs []int32 // Sorted IDs

func loadUnitsFromCSV(filename string) error {
    file, err := os.Open(filename)
    if err != nil {
        return err
    }
    defer file.Close()

    reader := csv.NewReader(file)
    records, err := reader.ReadAll()
    if err != nil {
        return err
    }

    for i, record := range records {
        if i == 0 {
            continue // Skip header row
        }
        if len(record) < 3 {
            continue
        }
        id, err := strconv.Atoi(record[0])
        if err != nil {
            fmt.Printf("⚠️ Failed to convert id (%s): %v\n", record[0], err)
            continue
        }
        unitID, err := strconv.Atoi(record[2])
        if err != nil {
            fmt.Printf("⚠️ Failed to convert unitID (%s): %v\n", record[2], err)
            continue
        }
        unitList[int32(id)] = Unit{
            Name: record[1],
            ID:   int32(unitID),
        }
        sortedIDs = append(sortedIDs, int32(id))
    }
    sort.Slice(sortedIDs, func(i, j int) bool {
        return sortedIDs[i] < sortedIDs[j]
    })
    return nil
}

func waitForGame(processName string) uint32 {
    for {
        pid, err := getProcessID(processName)
        if err == nil {
            return pid
        }
        time.Sleep(2 * time.Second) // Check every 2 seconds
    }
}

func main() {
    err := loadUnitsFromCSV("units.csv")
    if err != nil {
        fmt.Printf("Failed to load CSV: %v\n", err)
        return
    }

    processName := "vsac27_Release_CLIENT.exe"

    fmt.Println("🔍 Waiting for game process to start...")

    pid := waitForGame(processName) // Wait until the game starts

    if err != nil {
        fmt.Printf("Process not found: %v\n", err)
        return
    }

    handle := openProcess(pid)
    defer syscall.CloseHandle(handle)

    moduleBase, err := getModuleBaseAddress(pid, processName)
    if err != nil {
        fmt.Printf("Failed to get module base address: %v\n", err)
        return
    }

    basePointerAddr := moduleBase + basePointerOffset

    var baseAddr uintptr
    _, _, readErr := syscall.NewLazyDLL("kernel32.dll").NewProc("ReadProcessMemory").Call(
        uintptr(handle),
        basePointerAddr,
        uintptr(unsafe.Pointer(&baseAddr)),
        unsafe.Sizeof(baseAddr),
        0,
    )
    if readErr != syscall.Errno(0) {
        fmt.Printf("Failed to resolve base pointer: %v\n", readErr)
        return
    }

    targetAddr := baseAddr + fixedOffset
    reader := bufio.NewReader(os.Stdin)

    for {
        fmt.Println("==== Unit List (Sorted by ID) ====")
        for _, id := range sortedIDs {
            unit := unitList[id]
            fmt.Printf("%d: %s (%d)\n", id, unit.Name, unit.ID)
        }

        fmt.Print("Enter ID (Press TAB to stop writing and reselect): ")
        input, _ := reader.ReadString('\n')
        input = strings.TrimSpace(input)

        id64, err := strconv.ParseInt(input, 10, 32)
        if err != nil {
            fmt.Println("❌ Please enter a valid numeric ID.")
            continue
        }
        id := int32(id64)

        unit, exists := unitList[id]
        if !exists {
            fmt.Println("❌ The entered ID does not exist in the list.")
            continue
        }

        fmt.Printf("✅ %s (%d) Writing started...\n", unit.Name, unit.ID)

        stopChan := make(chan struct{})
        // Write loop (no logs during loop)
        go func(unitID int32, stop chan struct{}) {
            for {
                select {
                case <-stop:
                    return
                default:
                    syscall.NewLazyDLL("kernel32.dll").NewProc("WriteProcessMemory").Call(
                        uintptr(handle),
                        targetAddr,
                        uintptr(unsafe.Pointer(&unitID)),
                        unsafe.Sizeof(unitID),
                        0,
                    )
                    time.Sleep(1 * time.Second) // Write every second
                }
            }
        }(unit.ID, stopChan)

        fmt.Println("💡 Press TAB to stop writing and reselect.")

        for {
            char, _ := reader.ReadByte()
            if char == '\t' { // If TAB is pressed
                close(stopChan) // Stop writing
                fmt.Println("⏹ Writing stopped.")
                break
            }
        }
    }
}

func getProcessID(processName string) (uint32, error) {
    snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
    if err != nil {
        return 0, err
    }
    defer windows.CloseHandle(snapshot)

    var entry windows.ProcessEntry32
    entry.Size = uint32(unsafe.Sizeof(entry))
    err = windows.Process32First(snapshot, &entry)
    for err == nil {
        name := syscall.UTF16ToString(entry.ExeFile[:])
        if name == processName {
            return entry.ProcessID, nil
        }
        err = windows.Process32Next(snapshot, &entry)
    }
    return 0, fmt.Errorf("Process %s not found", processName)
}

func getModuleBaseAddress(pid uint32, moduleName string) (uintptr, error) {
    snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPMODULE|windows.TH32CS_SNAPMODULE32, pid)
    if err != nil {
        return 0, err
    }
    defer windows.CloseHandle(snapshot)

    var me windows.ModuleEntry32
    me.Size = uint32(unsafe.Sizeof(me))
    err = windows.Module32First(snapshot, &me)
    for err == nil {
        modName := syscall.UTF16ToString(me.Module[:])
        if modName == moduleName {
            return uintptr(me.ModBaseAddr), nil
        }
        err = windows.Module32Next(snapshot, &me)
    }
    return 0, fmt.Errorf("Module %s not found", moduleName)
}

func openProcess(pid uint32) syscall.Handle {
    handle, _, _ := syscall.NewLazyDLL("kernel32.dll").NewProc("OpenProcess").Call(
        uintptr(PROCESS_VM_READ|PROCESS_VM_WRITE|PROCESS_VM_OPERATION|PROCESS_QUERY_INFORMATION),
        0,
        uintptr(pid),
    )
    return syscall.Handle(handle)
}
