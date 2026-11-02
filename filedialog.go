package main

import (
	"image"
	"os"
	"os/exec"
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
		f, err := os.Open(path)
		if err != nil {
			return
		}
		defer f.Close()
		im, _, err := image.Decode(f)
		if err != nil {
			return
		}
		ch <- &droppedImage{name: strings.ToLower(path), b: pixBufFromImage(im)}
	}()
	return ch
}

func pickFilePath() string {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("osascript", "-e", `POSIX path of (choose file with prompt "Elige una imagen (PNG o JPG)" of type {"public.image"})`)
	case "windows":
		cmd = exec.Command("powershell", "-NoProfile", "-Command",
			`Add-Type -AssemblyName System.Windows.Forms; $f = New-Object System.Windows.Forms.OpenFileDialog; $f.Filter = 'Imágenes (PNG, JPG)|*.png;*.jpg;*.jpeg'; if ($f.ShowDialog() -eq 'OK') { $f.FileName }`)
	default:
		cmd = exec.Command("zenity", "--file-selection", "--title=Elige una imagen", "--file-filter=*.png *.jpg *.jpeg")
	}
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
