//go:build windows
// +build windows

package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	processName = "vsac27_Release_CLIENT.exe"

	// CEのスクショより
	baseRVA = uintptr(0x020023B8)
)

var (
	offsets = []uintptr{0x4A0, 0x108, 0x440, 0x188, 0x38, 0x534}
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("❌ Usage: ms-changer <unitValue>")
		return
	}
	unitValue, err := strconv.Atoi(os.Args[1])
	if err != nil {
		fmt.Println("❌ Invalid unitValue:", os.Args[1])
		return
	}

	fmt.Printf("✅ Writing unitValue: %d\n", unitValue)

	pid := waitForGame(processName)
	fmt.Println("🟢 Found PID:", pid)

	handle, err := openProcess(pid)
	fmt.Printf("🛠 OpenProcess handle: 0x%X\n", handle)
	if err != nil {
		fmt.Println("❌ openProcess error:", err)
		return
	}
	defer syscall.CloseHandle(handle)

	moduleBase, err := getModuleBaseAddress(pid, processName)
	if err != nil {
		fmt.Println("❌ getModuleBaseAddress failed:", err)
		return
	}
	fmt.Printf("🧩 Module base address: 0x%X\n", moduleBase)

	// Pointer chain from CE screenshot
	addr := moduleBase + baseRVA
	fmt.Printf("📌 Starting pointer chain from: 0x%X\n", addr)
	
	// Follow the pointer chain
	for i, offset := range offsets {
		var nextAddr uintptr
		ret, _, err := syscall.NewLazyDLL("kernel32.dll").NewProc("ReadProcessMemory").Call(
			uintptr(handle),
			addr,
			uintptr(unsafe.Pointer(&nextAddr)),
			unsafe.Sizeof(nextAddr),
			0,
		)
		if ret == 0 {
			fmt.Printf("❌ ReadProcessMemory failed at step %d (0x%X): %v\n", i+1, addr, err)
			return
		}
		addr = nextAddr + offset
		fmt.Printf("🔗 Step %d: 0x%X + 0x%X = 0x%X\n", i+1, nextAddr, offset, addr)
	}
	
	target := addr
	fmt.Printf("✏️ Final target address for writing: 0x%X\n", target)

	ret, _, err := syscall.NewLazyDLL("kernel32.dll").NewProc("WriteProcessMemory").Call(
		uintptr(handle),
		target,
		uintptr(unsafe.Pointer(&unitValue)),
		unsafe.Sizeof(unitValue),
		0,
	)
	if ret == 0 {
		fmt.Printf("❌ WriteProcessMemory failed: %v\n", err)
		return
	}
	fmt.Println("✅ Write successful.")
}

func waitForGame(name string) uint32 {
	for {
		pid, err := getProcessID(name)
		if err == nil {
			return pid
		}
		time.Sleep(1 * time.Second)
	}
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
		if strings.EqualFold(syscall.UTF16ToString(entry.ExeFile[:]), name) {
			return entry.ProcessID, nil
		}
		err = windows.Process32Next(snap, &entry)
	}
	return 0, fmt.Errorf("Process %s not found", name)
}

func getModuleBaseAddress(pid uint32, moduleName string) (uintptr, error) {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPMODULE|windows.TH32CS_SNAPMODULE32, pid)
	if err != nil {
		return 0, err
	}
	defer windows.CloseHandle(snap)

	var me windows.ModuleEntry32
	me.Size = uint32(unsafe.Sizeof(me))
	err = windows.Module32First(snap, &me)
	for err == nil {
		if strings.EqualFold(syscall.UTF16ToString(me.Module[:]), moduleName) {
			return uintptr(me.ModBaseAddr), nil
		}
		err = windows.Module32Next(snap, &me)
	}
	return 0, fmt.Errorf("Module %s not found", moduleName)
}

func openProcess(pid uint32) (syscall.Handle, error) {
	handleRaw, _, err := syscall.NewLazyDLL("kernel32.dll").NewProc("OpenProcess").Call(
		uintptr(0x0010|0x0020|0x0008|0x0400), // VM_READ | VM_WRITE | VM_OPERATION | QUERY_INFORMATION
		0,
		uintptr(pid),
	)
	handle := syscall.Handle(handleRaw)
	if handle == 0 || handle == syscall.InvalidHandle {
		return 0, fmt.Errorf("OpenProcess failed: %v", err)
	}
	return handle, nil
}
