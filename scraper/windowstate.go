package scraper

import (
	"fmt"
	"strings"
	"syscall"
	"unsafe"
)

const WINDOW_STRING_MAX = 512

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
var procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
var procGetWindowRect = user32.NewProc("GetWindowRect")
var procGetClientRect = user32.NewProc("GetClientRect")
var procGetClassName = user32.NewProc("GetClassNameW")
var procGetWindowText = user32.NewProc("GetWindowTextW")
var procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
var procSetFocus = user32.NewProc("SetFocus")

var procGetAncestor = user32.NewProc("GetAncestor")

const GA_ROOT = 2

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

type RECT struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

type WindowRect struct {
	X int
	Y int
	W int
	H int
}

type WindowMetrics struct {
	Hwnd      uintptr
	Top       uintptr
	PID       int
	State     string
	StateCode WindowState
	Pos       WindowRect
	MinPos    WindowRect
	Rect      WindowRect
	Client    WindowRect
	Content   WindowRect
}

func GetWindowMetrics(hwnd uintptr) (WindowMetrics, error) {
	metrics := WindowMetrics{Hwnd: hwnd}
	var placement WINDOWPLACEMENT
	placement.length = uint32(unsafe.Sizeof(placement))
	ret, _, err := procGetWindowPlacement.Call(hwnd, uintptr(unsafe.Pointer(&placement)))
	if ret == 0 {
		if isError(err) {
			return WindowMetrics{}, Fatal(err)
		}
	}

	metrics.State = WindowState(placement.showCmd).String()
	metrics.StateCode = WindowState(placement.showCmd)

	metrics.MinPos.X = int(placement.ptMinPosition.x)
	metrics.MinPos.Y = int(placement.ptMinPosition.y)

	metrics.Pos.X = int(placement.rect.left)
	metrics.Pos.Y = int(placement.rect.top)
	metrics.Pos.W = int(placement.rect.right - placement.rect.left)
	metrics.Pos.H = int(placement.rect.bottom - placement.rect.top)

	var wrect RECT
	ret, _, err = procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&wrect)))
	if ret == 0 {
		if isError(err) {
			return WindowMetrics{}, Fatal(err)
		}
	}
	metrics.Rect.X = int(wrect.Left)
	metrics.Rect.Y = int(wrect.Top)
	metrics.Rect.W = int(wrect.Right - wrect.Left)
	metrics.Rect.H = int(wrect.Bottom - wrect.Top)

	var crect RECT
	ret, _, err = procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&crect)))
	if ret == 0 {
		if isError(err) {
			return WindowMetrics{}, Fatal(err)
		}
	}
	metrics.Client.X = int(crect.Left)
	metrics.Client.Y = int(crect.Top)
	metrics.Client.W = int(crect.Right - crect.Left)
	metrics.Client.H = int(crect.Bottom - crect.Top)

	metrics.Content.X = int(wrect.Left + crect.Left)
	metrics.Content.Y = int(wrect.Top + crect.Top)
	metrics.Content.W = int(crect.Right - crect.Left)
	metrics.Content.H = int(crect.Bottom - crect.Top)

	metrics.Top, err = GetParentWindow(hwnd)
	if err != nil {
		return WindowMetrics{}, Fatal(err)
	}

	metrics.PID, err = GetWindowPid(metrics.Top)
	if err != nil {
		return WindowMetrics{}, Fatal(err)
	}

	return metrics, nil
}

func SetWindowFocus(hwnd uintptr) error {
	ret, _, err := procSetFocus.Call(hwnd)
	if ret == 0 {
		if isError(err) {
			return Fatal(err)
		}
	}
	return nil
}

func GetWindowPid(hwnd uintptr) (int, error) {
	var pid uint32
	tid, _, err := procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	if tid == 0 {
		if isError(err) {
			return 0, Fatal(err)
		}
	}
	return int(pid), nil
}

func GetWindowClass(hwnd uintptr) (string, error) {
	buf := make([]uint16, WINDOW_STRING_MAX)
	bufsiz := uintptr(len(buf)) * unsafe.Sizeof(buf[0])
	ret, _, err := procGetClassName.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), bufsiz)
	if ret == 0 {
		if isError(err) {
			return "", Fatal(err)
		}
	}
	return syscall.UTF16ToString(buf), nil
}

func GetWindowText(hwnd uintptr) (string, error) {
	buf := make([]uint16, WINDOW_STRING_MAX)
	bufsiz := uintptr(len(buf)) * unsafe.Sizeof(buf[0])
	ret, _, err := procGetWindowText.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), bufsiz)
	if ret == 0 {
		if isError(err) {
			return "", Fatal(err)
		}
	}
	return syscall.UTF16ToString(buf), nil
}

func GetWindowState(hwnd uintptr) (WindowState, error) {
	var placement WINDOWPLACEMENT
	placement.length = uint32(unsafe.Sizeof(placement))
	ret, _, err := procGetWindowPlacement.Call(hwnd, uintptr(unsafe.Pointer(&placement)))
	if ret == 0 {
		if isError(err) {
			return 0, Fatal(err)
		}
	}
	return WindowState(placement.showCmd), nil
}

func SetWindowState(hwnd uintptr, state WindowState) error {
	ret, _, err := procShowWindow.Call(hwnd, uintptr(state))
	if ret == 0 {
		if isError(err) {
			return Fatal(err)
		}
	}
	return nil
}

func SetForegroundWindow(hwnd uintptr) error {
	ret, _, err := procSetForegroundWindow.Call(hwnd)
	if ret == 0 {
		if isError(err) {
			return Fatal(err)
		}
	}
	return nil
}

func isError(err error) bool {
	if err == nil {
		return false
	}
	if strings.HasSuffix(fmt.Sprintf("%v", err), "The operation completed successfully.") {
		return false
	}
	return true
}

func GetParentWindow(hwnd uintptr) (uintptr, error) {
	top, _, err := procGetAncestor.Call(hwnd, GA_ROOT)
	if top == 0 {
		if isError(err) {
			return 0, Fatal(err)
		}
	}
	return uintptr(top), nil
}
