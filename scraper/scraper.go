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

const Version = "0.0.14"

type Scraper struct {
	Version            string
	pid                int
	GosseractVersion   string
	AvailableLanguages []string
	debug              bool
}

func NewScraper() (*Scraper, error) {
	s := Scraper{
		Version:          Version,
		GosseractVersion: gosseract.Version(),
		debug:            ViperGetBool("debug"),
	}
	var err error
	s.AvailableLanguages, err = gosseract.GetAvailableLanguages()
	if err != nil {
		return nil, Fatal(err)
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

	x, y, width, height := robotgo.GetClient(pid)
	if s.debug {
		log.Printf("WriteWindowImage: x=%d y=%d width=%d height=%d\n", x, y, width, height)
	}
	image, err := robotgo.CaptureImg(x, y, width, height)
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
