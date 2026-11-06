package scraper

import (
	"github.com/go-vgo/robotgo"
	"github.com/shirou/gopsutil/v4/process"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func start(t *testing.T, binary string, args ...string) int {
	cmd := exec.Command(binary, args...)
	log.Printf("start: %v\n", cmd)
	err := cmd.Start()
	require.Nil(t, err)
	time.Sleep(1 * time.Second)
	return cmd.Process.Pid
}

func pslist(t *testing.T, args ...string) error {
	cmd := exec.Command("pslist", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.Printf("pslist: %v\n", cmd)
	err := cmd.Run()
	log.Printf("pslist err: %v\n", err)
	return err
}
func pskill(t *testing.T, args ...string) {
	cmd := exec.Command("pskill", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.Printf("pskill: %v\n", cmd)
	err := cmd.Run()
	require.Nil(t, err)
}

func initTestConfig(t *testing.T) *Scraper {
	Init("bot", Version, filepath.Join("testdata", "config.yaml"))
	viper.SetConfigFile("testdata/config.yaml")
	err := viper.ReadInConfig()
	require.Nil(t, err)
	pattern := ViperGetString("process_pattern")
	log.Printf("pattern=%s\n", pattern)
	if pslist(t, pattern) == nil {
		log.Printf("killing: %s\n", pattern)
		pskill(t, pattern)
	} else {
		log.Printf("%s is not running\n", pattern)
	}
	log.Printf("starting: %s\n", pattern)
	pid := start(t, pattern)
	proc, err := process.NewProcess(int32(pid))
	require.Nil(t, err)
	procName, err := proc.Name()
	require.Nil(t, err)
	log.Printf("pid=%d name=%s\n", pid, procName)
	scraper, err := NewScraper(pid)
	require.Nil(t, err)
	return scraper
}

func TestScraperInit(t *testing.T) {
	scraper := initTestConfig(t)
	require.NotNil(t, scraper)
	err := scraper.Activate()
	require.Nil(t, err)
	err = robotgo.TypeStr("foo\nmoo\ngoo\n")
	require.Nil(t, err)
	pngFile := filepath.Join("testdata", "scraper.png")
	err = scraper.CapturePng(pngFile)
	require.Nil(t, err)

	text, err := scraper.PngToText(pngFile)
	require.Nil(t, err)
	log.Printf("text: %s\n", text)
}
