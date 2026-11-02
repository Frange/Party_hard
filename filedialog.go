package main

import (
	"errors"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// pickImageAsync opens the system's "choose a file" dialog without freezing the
// editor and delivers the chosen picture (or nothing if it was cancelled).
func pickImageAsync() chan *droppedImage {
	ch := make(chan *droppedImage, 1)
	go func() {
		defer close(ch)
		path := pickFilePath()
		if path == "" {
			return
		}
		b, err := loadImageAny(path)
		if err != nil {
			ch <- &droppedImage{err: err.Error()}
			return
		}
		ch <- &droppedImage{name: strings.ToLower(path), b: b}
	}()
	return ch
}

func pickFilePath() string {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("osascript", "-e", `POSIX path of (choose file with prompt "Elige una imagen (PNG, JPG o PDF)" of type {"public.image", "com.adobe.pdf"})`)
	case "windows":
		cmd = exec.Command("powershell", "-NoProfile", "-Command",
			`Add-Type -AssemblyName System.Windows.Forms; $f = New-Object System.Windows.Forms.OpenFileDialog; $f.Filter = 'Imágenes (PNG, JPG, PDF)|*.png;*.jpg;*.jpeg;*.pdf'; if ($f.ShowDialog() -eq 'OK') { $f.FileName }`)
	default:
		cmd = exec.Command("zenity", "--file-selection", "--title=Elige una imagen", "--file-filter=*.png *.jpg *.jpeg *.pdf")
	}
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// loadImageAny reads a PNG, a JPG or the first page of a PDF.
func loadImageAny(path string) (*pixBuf, error) {
	if strings.HasSuffix(strings.ToLower(path), ".pdf") {
		png, err := pdfToPNG(path)
		if err != nil {
			return nil, err
		}
		defer os.Remove(png)
		path = png
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	im, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}
	return pixBufFromImage(im), nil
}

// pdfToPNG renders the first page of a PDF with the tools the system already has.
func pdfToPNG(path string) (string, error) {
	tmp, err := os.MkdirTemp("", "ph-pdf")
	if err != nil {
		return "", err
	}
	out := filepath.Join(tmp, "page.png")
	switch runtime.GOOS {
	case "darwin":
		err = exec.Command("sips", "-s", "format", "png", path, "--out", out).Run()
	case "windows":
		return "", errors.New("En Windows no se pueden abrir PDF: expórtalo antes a PNG o JPG")
	default:
		err = exec.Command("pdftoppm", "-png", "-singlefile", "-r", "150", path, filepath.Join(tmp, "page")).Run()
	}
	if err != nil {
		return "", errors.New("No se pudo leer el PDF: expórtalo a PNG o JPG")
	}
	return out, nil
}
