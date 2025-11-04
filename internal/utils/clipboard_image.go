package utils

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"golang.design/x/clipboard"
)

// ClipboardImageHandler handles clipboard image operations
type ClipboardImageHandler struct {
	initialized bool
}

// NewClipboardImageHandler creates a new clipboard image handler
func NewClipboardImageHandler() *ClipboardImageHandler {
	return &ClipboardImageHandler{}
}

// Initialize initializes the clipboard
func (h *ClipboardImageHandler) Initialize() error {
	if h.initialized {
		return nil
	}

	err := clipboard.Init()
	if err != nil {
		return fmt.Errorf("failed to initialize clipboard: %w\n\n%s", err, getClipboardHelp())
	}

	h.initialized = true
	return nil
}

// HasImage checks if the clipboard contains an image
func (h *ClipboardImageHandler) HasImage() bool {
	if !h.initialized {
		// Try to initialize if not already done
		if err := h.Initialize(); err != nil {
			return false
		}
	}

	data := clipboard.Read(clipboard.FmtImage)
	return len(data) > 0
}

// SaveClipboardImage saves the clipboard image to the specified directory
// Returns the relative path to the saved image
func (h *ClipboardImageHandler) SaveClipboardImage(attachmentsDir, baseName string) (string, error) {
	if !h.initialized {
		if err := h.Initialize(); err != nil {
			return "", err
		}
	}

	// Read image from clipboard
	data := clipboard.Read(clipboard.FmtImage)
	if len(data) == 0 {
		return "", fmt.Errorf("no image data in clipboard")
	}

	// Decode the image
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("failed to decode clipboard image: %w", err)
	}

	// Create attachments directory if it doesn't exist
	if err := os.MkdirAll(attachmentsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create attachments directory: %w", err)
	}

	// Generate filename with timestamp
	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("%s-%s.png", baseName, timestamp)
	imagePath := filepath.Join(attachmentsDir, filename)

	// Create the file
	file, err := os.Create(imagePath)
	if err != nil {
		return "", fmt.Errorf("failed to create image file: %w", err)
	}
	defer file.Close()

	// Encode and save as PNG
	if err := png.Encode(file, img); err != nil {
		return "", fmt.Errorf("failed to encode image: %w", err)
	}

	return filename, nil
}

// getClipboardHelp returns platform-specific help for clipboard issues
func getClipboardHelp() string {
	switch runtime.GOOS {
	case "linux":
		return `On Linux, clipboard support requires X11 or Wayland display server and clipboard utilities.

For X11, ensure you have one of these installed:
  - xclip: sudo apt-get install xclip (Debian/Ubuntu) or sudo yum install xclip (RedHat/Fedora)
  - xsel: sudo apt-get install xsel (Debian/Ubuntu) or sudo yum install xsel (RedHat/Fedora)

For Wayland, ensure you have:
  - wl-clipboard: sudo apt-get install wl-clipboard (Debian/Ubuntu)

Also ensure DISPLAY environment variable is set (for X11) or you're running in a Wayland session.`

	case "darwin":
		return `On macOS, clipboard support should work out of the box.
If you're experiencing issues, ensure you're running macOS 10.10 or later.`

	case "windows":
		return `On Windows, clipboard support should work out of the box.
If you're experiencing issues, ensure you're running Windows 7 or later.`

	default:
		return fmt.Sprintf("Clipboard support may not be available on %s", runtime.GOOS)
	}
}
