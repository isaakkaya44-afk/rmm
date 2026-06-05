package remote

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type RustDeskInfo struct {
	ID       string `json:"rustdesk_id"`
	Password string `json:"rustdesk_password,omitempty"`
	Running  bool   `json:"rustdesk_running"`
}

type Detector struct {
	rustdeskPath string
}

func NewDetector(path string) *Detector {
	return &Detector{rustdeskPath: path}
}

func (d *Detector) Detect() *RustDeskInfo {
	info := &RustDeskInfo{}

	if d.rustdeskPath == "" {
		info.Running = d.isRunning()
		if info.Running {
			info.ID = d.readConfigID()
		}
		return info
	}

	idPath := filepath.Join(d.rustdeskPath, "config", "RustDesk.toml")
	info.Running = d.isRunning()

	if data, err := os.ReadFile(idPath); err == nil {
		re := regexp.MustCompile(`id\s*=\s*"([^"]+)"`)
		matches := re.FindStringSubmatch(string(data))
		if len(matches) > 1 {
			info.ID = matches[1]
		}

		rePwd := regexp.MustCompile(`password\s*=\s*"([^"]+)"`)
		pwdMatches := rePwd.FindStringSubmatch(string(data))
		if len(pwdMatches) > 1 {
			info.Password = pwdMatches[1]
		}
	}

	if info.ID == "" && info.Running {
		info.ID = d.readConfigID()
	}

	return info
}

func (d *Detector) isRunning() bool {
	cmd := exec.Command("tasklist", "/FI", "IMAGENAME eq RustDesk.exe")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(output), "RustDesk.exe")
}

func (d *Detector) readConfigID() string {
	configDirs := []string{
		os.Getenv("APPDATA") + "\\RustDesk\\config\\RustDesk.toml",
		os.Getenv("LOCALAPPDATA") + "\\RustDesk\\config\\RustDesk.toml",
		"C:\\Program Files\\RustDesk\\config\\RustDesk.toml",
	}

	for _, path := range configDirs {
		if data, err := os.ReadFile(path); err == nil {
			re := regexp.MustCompile(`id\s*=\s*"([^"]+)"`)
			matches := re.FindStringSubmatch(string(data))
			if len(matches) > 1 {
				return matches[1]
			}
		}
	}
	return ""
}

func ExecuteCommand(command string) (string, error) {
	cmd := exec.Command("powershell", "-NoProfile", "-Command", command)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return string(exitErr.Stderr), fmt.Errorf("command failed: %w", err)
		}
		return "", err
	}
	return string(output), nil
}
