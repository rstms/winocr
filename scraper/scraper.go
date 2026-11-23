package scraper

import (
	"github.com/go-vgo/robotgo"
	"github.com/otiai10/gosseract/v2"
	"image/png"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

const Version = "0.0.15"

type Display struct {
	Index  int
	Rect   robotgo.Rect
	Bounds robotgo.Rect
	Screen robotgo.Rect
	Scale  robotgo.Size
}

type Client struct {
	PID      int
	Position robotgo.Rect
	Bounds   robotgo.Rect
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
}

func NewScraper() (*Scraper, error) {
	s := Scraper{
		Version:          Version,
		GosseractVersion: gosseract.Version(),
		debug:            ViperGetBool("debug"),
		Displays:         []Display{},
	}
	s.ScreenSizeX, s.ScreenSizeY = robotgo.GetScreenSize()

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
	return nil
}

func (s *Scraper) WindowText(pid int) (string, error) {
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
	err = s.WriteWindowImage(ofp, pid)
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

func (s *Scraper) WriteWindowImage(writer io.Writer, pid int) error {

	activeName, activeExe, activePid := robotgo.GetActiveApp()
	if s.debug {
		log.Printf("WriteWindowImage: activeName=%s activeExe=%s activePid=%d\n", activeName, activeExe, activePid)
	}

	err := s.ActivateWindow(pid)
	if err != nil {
		return Fatal(err)
	}

	defer func(pid int) {
		err := s.ActivateWindow(pid)
		if err != nil {
			log.Fatal(Fatal(err))
			os.Exit(1)
		}
	}(activePid)

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
		log.Printf("WriteWindowImage: %s\n", FormatJSON(s))
		log.Printf("CaptureImg(%d, %d, %d, %d)\n", x, y, w, h)
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
			return nil
		}
	}
	return Fatalf("PID %d not found", pid)
}
