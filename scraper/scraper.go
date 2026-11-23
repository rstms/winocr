package scraper

import (
	"fmt"
	"github.com/go-vgo/robotgo"
	"github.com/otiai10/gosseract/v2"
	"image/png"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unsafe"
)

const Version = "0.1.6"

type Display struct {
	Index  int
	Rect   robotgo.Rect
	Bounds robotgo.Rect
	Screen robotgo.Rect
	Scale  robotgo.Size
}

type Client struct {
	PID        int
	Position   robotgo.Rect
	Bounds     robotgo.Rect
	State      WindowState
	hwnd       uintptr
	SavedState bool
}

type Scraper struct {
	Version            string
	pid                int
	GosseractVersion   string
	AvailableLanguages []string
	debug              bool
	ScreenSizeX        int
	ScreenSizeY        int
	Displays           []Display
	Client             Client
	ActivePid          int
	ActiveName         string
	ActiveExe          string
}

type Window struct {
	Name    string
	Title   string
	Class   string
	Handle  string
	PID     int
	Hwnd    uintptr
	Metrics WindowMetrics
}

func NewScraper() (*Scraper, error) {
	s := Scraper{
		Version:          Version,
		GosseractVersion: gosseract.Version(),
		debug:            ViperGetBool("debug"),
		Displays:         []Display{},
	}
	s.ScreenSizeX, s.ScreenSizeY = robotgo.GetScreenSize()
	s.ActiveName, s.ActiveExe, s.ActivePid = robotgo.GetActiveApp()

	var err error
	s.AvailableLanguages, err = gosseract.GetAvailableLanguages()
	if err != nil {
		return nil, Fatal(err)
	}
	displayCount := robotgo.DisplaysNum()
	for i := 0; i < displayCount; i++ {
		d := Display{Index: i}
		d.Rect = robotgo.GetDisplayRect(i)
		d.Bounds.Point.X, d.Bounds.Point.Y, d.Bounds.Size.W, d.Bounds.Size.H = robotgo.GetDisplayBounds(i)
		d.Screen = robotgo.GetScreenRect(i)
		d.Scale.W, d.Scale.H = robotgo.GetScaleSize(i)
		s.Displays = append(s.Displays, d)
	}
	if s.debug {
		log.Printf("NewScraper: %s\n", FormatJSON(s))
	}
	return &s, nil

}

func (s *Scraper) Close() error {
	if s.Client.SavedState {
		err := SetWindowState(s.Client.hwnd, s.Client.State)
		if err != nil {
			return Fatal(err)
		}
	}
	if s.ActivePid != 0 {
		err := robotgo.ActivePid(s.ActivePid)
		if err != nil {
			return Fatal(err)
		}
	}
	return nil
}

func (s *Scraper) WindowText(id string) (string, error) {
	tempDir, err := os.MkdirTemp("", "image*")
	if err != nil {
		return "", Fatal(err)
	}
	defer os.RemoveAll(tempDir)
	filename := filepath.Join(tempDir, "image.png")
	ofp, err := os.Create(filename)
	if err != nil {
		return "", Fatal(err)
	}
	defer ofp.Close()
	err = s.WriteWindowImage(ofp, id)
	if err != nil {
		return "", Fatal(err)
	}

	dataDir, ok := os.LookupEnv("TESSDATA_PREFIX")
	if !ok {
		return "", Fatalf("unset: TESSDATA_PREFIX")
	}
	if !IsDir(dataDir) {
		return "", Fatalf("TESSDATA_PREFIX value not a directory")
	}

	if s.debug {
		log.Printf("opening gosseract client...\n")
	}

	client := gosseract.NewClient()

	defer client.Close()
	if s.debug {
		log.Printf("gosseract client version: %s\n", client.Version())
	}

	client.SetImage(filename)

	text, err := client.Text()
	if err != nil {
		return "", Fatal(err)
	}
	return text, nil
}

func (s *Scraper) WriteWindowImage(writer io.Writer, id string) error {

	x, y, w, h, err := s.getCaptureRect(id)
	if err != nil {
		return Fatal(err)
	}

	image, err := robotgo.CaptureImg(x, y, w, h)
	if err != nil {
		return Fatal(err)
	}
	err = png.Encode(writer, image)
	if err != nil {
		return Fatal(err)
	}
	return nil
}

func (s *Scraper) MapWindows() (map[string]Window, error) {

	windows := map[string]Window{}

	names, err := robotgo.FindNames()
	if err != nil {
		return nil, Fatal(err)
	}
	for _, name := range names {
		if name != "" {
			ids, err := robotgo.FindIds(name)
			if err != nil {
				return nil, Fatal(err)
			}
			for _, id := range ids {
				hwnd := robotgo.GetHWNDByPid(id)
				if hwnd != 0 {
					handle := fmt.Sprintf("0x%x", hwnd)
					_, ok := windows[handle]
					if !ok {
						w := Window{
							Name:   name,
							Handle: handle,
							Hwnd:   uintptr(hwnd),
							PID:    id,
						}
						w.Title, err = GetWindowText(uintptr(hwnd))
						if err != nil {
							return nil, Fatal(err)
						}

						w.Class, err = GetWindowClass(uintptr(hwnd))
						if err != nil {
							return nil, Fatal(err)
						}

						w.Metrics, err = GetWindowMetrics(uintptr(hwnd))
						if err != nil {
							return nil, Fatal(err)
						}
						windows[handle] = w
					}
				}
			}
		}
	}
	return windows, nil
}

func (s *Scraper) ActivateWindow(pid int) error {
	pids, err := robotgo.Pids()
	if err != nil {
		return Fatal(err)
	}
	for _, p := range pids {
		if p == pid {

			if s.debug {
				log.Printf("activating PID %d\n", pid)
			}

			err = robotgo.ActivePid(pid)
			if err != nil {
				return Fatal(err)
			}
			hwndByPid := robotgo.GetHWNDByPid(pid)
			if s.debug {
				log.Printf("pid %d GetHWNDByPid hwnd=%v\n", pid, hwndByPid)
			}

			hwndGet := robotgo.GetHWND()
			if s.debug {
				log.Printf("pid %d GetHWND hwnd=%v\n", pid, hwndGet)
			}

			handle := robotgo.GetActive()
			s.Client.hwnd = uintptr(unsafe.Pointer(handle.HWnd))
			if s.debug {
				log.Printf("pid %d GetActive hwnd=%v\n", pid, s.Client.hwnd)
			}

			s.Client.State, err = GetWindowState(s.Client.hwnd)
			if err != nil {
				return Fatal(err)
			}
			s.Client.SavedState = true
			if s.debug {
				log.Printf("pid %d window state is %v\n", pid, s.Client.State)
			}
			if s.Client.State == WindowStateMinimized || s.Client.State == WindowStateHidden {
				newState := WindowStateRestore
				if s.debug {
					log.Printf("setting pid %d window state to %v\n", pid, newState)
				}
				err := SetWindowState(s.Client.hwnd, newState)
				if err != nil {
					return Fatal(err)
				}
				time.Sleep(time.Second)
				readbackState, err := GetWindowState(s.Client.hwnd)
				if err != nil {
					return Fatal(err)
				}
				if s.debug {
					log.Printf("pid %d readback state: %v\n", pid, readbackState)
				}
				if readbackState != WindowStateNormal {
					return Fatalf("abnormal window state: %v", readbackState)
				}
			}

			err = robotgo.ActivePid(pid)
			if err != nil {
				return Fatal(err)
			}

			err = SetForegroundWindow(s.Client.hwnd)
			if err != nil {
				return Fatal(err)
			}

			robotgo.SetForeg(robotgo.GetHWND())
			robotgo.SetFocus(robotgo.GetHWND())

			return nil
		}
	}
	return Fatalf("PID %d not found", pid)
}

func (s *Scraper) getCaptureRect(id string) (int, int, int, int, error) {

	id = strings.ToLower(id)
	var pid int
	var hwnd uintptr
	switch {
	case id == "root":
		return 0, 0, s.ScreenSizeX, s.ScreenSizeY, nil
	case regexp.MustCompile(`^0x[0-9a-f]+$`).MatchString(id):
		handle, err := strconv.ParseUint(id[2:], 16, 64)
		if err != nil {
			return 0, 0, 0, 0, Fatal(err)
		}
		metrics, err := GetWindowMetrics(uintptr(handle))
		if err != nil {
			return 0, 0, 0, 0, Fatal(err)
		}
		log.Printf("found: id=%s metrics=%s\n", id, FormatJSON(metrics))
		pid = metrics.PID
		hwnd = metrics.Hwnd
		/*
			windows, err := s.MapWindows()
			if err != nil {
				return 0, 0, 0, 0, Fatal(err)
			}
			for handle, w := range windows {
				if strings.ToLower(handle) == id {
					window = w
					found = true
					break
				}
			}
		*/

	case regexp.MustCompile(`^[0-9]+$`).MatchString(id):
		windows, err := s.MapWindows()
		if err != nil {
			return 0, 0, 0, 0, Fatal(err)
		}
		pid, err = strconv.Atoi(id)
		if err != nil {
			return 0, 0, 0, 0, Fatal(err)
		}
		for _, w := range windows {
			if w.PID == pid {
				hwnd = w.Hwnd
				break
			}
		}

	/*
		pid, err := strconv.Atoi(id)
		if err != nil {
			return 0, 0, 0, 0, Fatal(err)
		}
		err = s.ActivateWindow(pid)
		if err != nil {
			return 0, 0, 0, 0, Fatal(err)
		}
		time.Sleep(1 * time.Second)

		s.Client.Position.Point.X, s.Client.Position.Point.Y, s.Client.Position.Size.W, s.Client.Position.Size.H = robotgo.GetClient(pid)
		s.Client.Bounds.Point.X, s.Client.Bounds.Point.Y, s.Client.Bounds.Size.W, s.Client.Bounds.Size.H = robotgo.GetBounds(pid)

		xscale := float64(s.Displays[0].Bounds.Size.W) / float64(s.Displays[0].Scale.W)
		yscale := float64(s.Displays[0].Bounds.Size.H) / float64(s.Displays[0].Scale.H)

		x := int(float64(s.Client.Position.Point.X) * xscale)
		y := int(float64(s.Client.Position.Point.Y) * yscale)
		w := int(float64(s.Client.Position.Size.W) * xscale)
		h := int(float64(s.Client.Position.Size.H) * yscale)

		if s.debug {
			log.Printf("Client: %s\n", FormatJSON(s.Client))
			log.Printf("CaptureImg(%d, %d, %d, %d)\n", x, y, w, h)
		}
		return x, y, w, h, nil
	*/

	default:
		return 0, 0, 0, 0, Fatalf("unexpected id format: %s", id)
	}
	if pid == 0 {
		return 0, 0, 0, 0, Fatalf("id not found: %s", id)
	}

	err := s.ActivateWindow(pid)
	if err != nil {
		return 0, 0, 0, 0, Fatal(err)
	}

	metrics, err := GetWindowMetrics(hwnd)
	if err != nil {
		return 0, 0, 0, 0, Fatal(err)
	}
	return metrics.Content.X, metrics.Content.Y, metrics.Content.W, metrics.Content.H, nil
}
