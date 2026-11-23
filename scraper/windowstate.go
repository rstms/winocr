package scraper

import (
	"syscall"
	"unsafe"
)

type WindowState uint32

const (
	WindowStateHidden WindowState = iota
	WindowStateNormal
	WindowStateMinimized
	WindowStateMaximized
	WindowStateShowNoActivate
	WindowStateShow
	WindowStateMinimizeNoActivate
	WindowStateShowDefault
	WindowStateForceMinimize
	WindowStateRestore
)

func (s WindowState) String() string {
	return []string{
		"WindowStateHidden",
		"WindowStateNormal",
		"WindowStateMinimized",
		"WindowStateMaximized",
		"WindowStateShowNoActivate",
		"WindowStateShow",
		"WindowStateMinimizeNoActivate",
		"WindowStateShowDefault",
		"WindowStateForceMinimize",
		"WindowStateRestore",
	}[s]
}

var user32 = syscall.NewLazyDLL("user32.dll")
var procGetWindowPlacement = user32.NewProc("GetWindowPlacement")
var procShowWindow = user32.NewProc("ShowWindow")

type WINDOWPLACEMENT struct {
	length        uint32
	flags         uint32
	showCmd       uint32
	ptMinPosition struct {
		x int32
		y int32
	}
	rect struct {
		left   int32
		top    int32
		right  int32
		bottom int32
	}
}

func GetWindowState(hwnd uintptr) (WindowState, error) {
	var placement WINDOWPLACEMENT
	placement.length = uint32(unsafe.Sizeof(placement))
	ret, _, _ := procGetWindowPlacement.Call(hwnd, uintptr(unsafe.Pointer(&placement)))
	if ret == 0 {
		errorCode := syscall.GetLastError()
		return 0, Fatalf("GetWindowPlacement failed with code: %d", errorCode)
	}
	return WindowState(placement.showCmd), nil
}

func SetWindowState(hwnd uintptr, state WindowState) error {
	ret, _, _ := procShowWindow.Call(hwnd, uintptr(state))
	if ret == 0 {
		errorCode := syscall.GetLastError()
		return Fatalf("ShowWindow failed with code: %d", errorCode)
	}
	return nil
}
