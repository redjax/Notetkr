package utils

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"runtime"

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

// SaveClipboardImage saves the clipboard image to a centralized attachments directory
// Returns just the filename (since all images are in the same imgs directory)
// If an identical image already exists, returns the existing filename
func (h *ClipboardImageHandler) SaveClipboardImage(imgsDir, baseName string) (string, error) {
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

	// Encode image to bytes for hashing
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", fmt.Errorf("failed to encode image: %w", err)
	}
	imageBytes := buf.Bytes()

	// Calculate SHA256 hash of the image
	hash := sha256.Sum256(imageBytes)
	hashString := hex.EncodeToString(hash[:])

	// Generate filename with hash prefix
	filename := fmt.Sprintf("%s-%s.png", baseName, hashString[:12])
	imagePath := filepath.Join(imgsDir, filename)

	// Check if file already exists (by name/hash)
	if _, err := os.Stat(imagePath); err == nil {
		// File already exists, return existing filename
		return filename, nil
	}

	// Create imgs directory if it doesn't exist
	if err := os.MkdirAll(imgsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create imgs directory: %w", err)
	}

	// Create the file
	file, err := os.Create(imagePath)
	if err != nil {
		return "", fmt.Errorf("failed to create image file: %w", err)
	}
	defer file.Close()

	// Write the already-encoded image bytes
	if _, err := file.Write(imageBytes); err != nil {
		return "", fmt.Errorf("failed to write image: %w", err)
	}

	return filename, nil
}

// findImageByHashGlobal searches for an existing image file with the given hash
// across all .attachments directories in the root directory
func findImageByHashGlobal(rootDir, hash string) (string, error) {
	hashPrefix := hash[:12]
	var foundFile string

	// Walk through all directories looking for .attachments folders
	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Only look in .attachments directories
		if !info.IsDir() || filepath.Base(path) != ".attachments" {
			return nil
		}

		// Search for images with matching hash in this .attachments directory
		filepath.Walk(path, func(imgPath string, imgInfo os.FileInfo, err error) error {
			if err != nil || imgInfo.IsDir() {
				return nil
			}

			// Check if this is a PNG file with matching hash
			if filepath.Ext(imgPath) == ".png" {
				filename := filepath.Base(imgPath)
				// Check if filename contains the hash prefix (format: image-HASH.png)
				if len(filename) >= 16 {
					// Extract the hash part (after "image-" and before ".png")
					parts := filename[len("image-"):]
					if len(parts) >= 12 && parts[:12] == hashPrefix {
						foundFile = imgPath
						return filepath.SkipDir // Stop searching once found
					}
				}
			}
			return nil
		})

		if foundFile != "" {
			return filepath.SkipDir // Stop outer walk if we found the file
		}

		return nil
	})

	if foundFile != "" {
		return foundFile, nil
	}

	if err != nil {
		return "", err
	}

	return "", fmt.Errorf("no matching image found")
}

// findImageByHash searches for an existing image file with the given hash
func findImageByHash(dir, hash string) (string, error) {
	// List all PNG files in the directory
	files, err := filepath.Glob(filepath.Join(dir, "*.png"))
	if err != nil {
		return "", err
	}

	// Check each file for the hash in the filename
	hashPrefix := hash[:12]
	for _, file := range files {
		filename := filepath.Base(file)
		if len(filename) >= 12 && filename[len(filename)-16:len(filename)-4] == hashPrefix {
			return file, nil
		}
	}

	return "", fmt.Errorf("no matching image found")
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
