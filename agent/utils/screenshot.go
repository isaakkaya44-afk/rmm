package utils

import (
	"encoding/base64"
	"fmt"
	"os/exec"
	"strings"
)

func CaptureScreen() (string, error) {
	psScript := `
Add-Type -AssemblyName System.Drawing
$screen = [System.Windows.Forms.Screen]::PrimaryScreen.Bounds
$bitmap = New-Object System.Drawing.Bitmap $screen.Width, $screen.Height
$graphics = [System.Drawing.Graphics]::FromImage($bitmap)
$graphics.CopyFromScreen($screen.X, $screen.Y, 0, 0, $screen.Size)
$ms = New-Object System.IO.MemoryStream
$bitmap.Save($ms, [System.Drawing.Imaging.ImageFormat]::Png)
$base64 = [System.Convert]::ToBase64String($ms.ToArray())
$graphics.Dispose()
$bitmap.Dispose()
Write-Output $base64
`
	cmd := exec.Command("powershell", "-NoProfile", "-Command", psScript)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("screenshot failed: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

type ScreenshotResult struct {
	Base64 string `json:"base64"`
	Size   int    `json:"size"`
}

func CaptureCompressed() (*ScreenshotResult, error) {
	base64Str, err := CaptureScreen()
	if err != nil {
		return nil, err
	}

	data, err := base64.StdEncoding.DecodeString(base64Str)
	if err != nil {
		return nil, err
	}

	return &ScreenshotResult{
		Base64: base64Str,
		Size:   len(data),
	}, nil
}
