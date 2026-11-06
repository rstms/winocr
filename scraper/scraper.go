package scraper

import (
	"github.com/go-vgo/robotgo"
	"github.com/otiai10/gosseract/v2"
	"image/png"
	"io"
	"log"
	"os"
	"path/filepath"
)

type Scraper struct {
	pid int
}

func NewScraper() *Scraper {
	return &Scraper{}
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
	client := gosseract.NewClient()
	defer client.Close()
	client.SetImage(filename)
	text, err := client.Text()
	if err != nil {
		return "", Fatal(err)
	}
	return text, nil
}

func (s *Scraper) WriteWindowImage(writer io.Writer, pid int) error {

	err := s.ActivateWindow(pid)
	if err != nil {
		return Fatal(err)
	}
	x, y, width, height := robotgo.GetClient(pid)
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
			log.Printf("activating PID %d\n", pid)
			err = robotgo.ActivePid(pid)
			if err != nil {
				return Fatal(err)
			}
			return nil
		}
	}
	return Fatalf("PID %d not found", pid)
}
