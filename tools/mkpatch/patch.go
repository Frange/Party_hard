package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Diff: what changes from the base manifest to the new one.
type Diff struct {
	Added, Changed, Removed []string // relative paths, sorted
	AddedDirs               []string // new empty folders
}

// computeDiff compares two manifests. A file is "changed" when its content or its permissions differ.
// On Mac, if anything inside an .app changes, its whole signature (executable, Info.plist and
// _CodeSignature/CodeResources) travels with it so the ad hoc seal stays coherent.
func computeDiff(base, next Manifest) Diff {
	old := map[string]Entry{}
	for _, e := range base.Files {
		old[e.Path] = e
	}
	now := map[string]Entry{}
	for _, e := range next.Files {
		now[e.Path] = e
	}
	var d Diff
	touchedApps := map[string]bool{}
	for _, e := range next.Files {
		o, ok := old[e.Path]
		switch {
		case !ok:
			d.Added = append(d.Added, e.Path)
		case o.SHA256 != e.SHA256 || o.Size != e.Size || o.Mode != e.Mode:
			d.Changed = append(d.Changed, e.Path)
		default:
			continue
		}
		if app := appOf(e.Path); app != "" {
			touchedApps[app] = true
		}
	}
	for _, e := range base.Files {
		if _, ok := now[e.Path]; !ok {
			d.Removed = append(d.Removed, e.Path)
			if app := appOf(e.Path); app != "" {
				touchedApps[app] = true
			}
		}
	}
	in := map[string]bool{}
	for _, p := range append(append([]string{}, d.Added...), d.Changed...) {
		in[p] = true
	}
	for app := range touchedApps {
		for _, e := range next.Files {
			if in[e.Path] || !strings.HasPrefix(e.Path, app+"/Contents/") {
				continue
			}
			rest := strings.TrimPrefix(e.Path, app+"/Contents/")
			if rest == "Info.plist" || rest == "_CodeSignature/CodeResources" || strings.HasPrefix(rest, "MacOS/") {
				d.Changed = append(d.Changed, e.Path)
				in[e.Path] = true
			}
		}
	}
	oldDirs := map[string]bool{}
	for _, x := range base.EmptyDirs {
		oldDirs[x] = true
	}
	for _, x := range next.EmptyDirs {
		if !oldDirs[x] {
			d.AddedDirs = append(d.AddedDirs, x)
		}
	}
	sort.Strings(d.Added)
	sort.Strings(d.Changed)
	sort.Strings(d.Removed)
	sort.Strings(d.AddedDirs)
	return d
}

// appOf: "PartyHardy.app/Contents/MacOS/PartyHardy" -> "PartyHardy.app" ("" outside a bundle).
func appOf(p string) string {
	first := strings.SplitN(p, "/", 2)[0]
	if strings.HasSuffix(first, ".app") && strings.Contains(p, "/") {
		return first
	}
	return ""
}

func parseSemver(v string) ([3]int, bool) {
	var r [3]int
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return r, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return r, false
		}
		r[i] = n
	}
	return r, true
}

func semverLess(a, b string) bool {
	x, _ := parseSemver(a)
	y, _ := parseSemver(b)
	for i := 0; i < 3; i++ {
		if x[i] != y[i] {
			return x[i] < y[i]
		}
	}
	return false
}

// changelogBetween: the CHANGELOG.md entries with from < version <= to, as written.
func changelogBetween(changelog, from, to string) string {
	var out []string
	keep := false
	for _, line := range strings.Split(strings.ReplaceAll(changelog, "\r\n", "\n"), "\n") {
		if strings.HasPrefix(line, "## [v") {
			v := strings.SplitN(strings.TrimPrefix(line, "## [v"), "]", 2)[0]
			keep = semverLess(from, v) && !semverLess(to, v)
		}
		if keep && strings.TrimSpace(line) != "---" {
			out = append(out, line)
		}
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

func platformName(p string) string {
	if p == "mac" {
		return "Mac"
	}
	return "Windows"
}

func patchFileName(platform, from, to string) string {
	return fmt.Sprintf("PartyHardy-Parche-%s-%s-a-%s.zip", platformName(platform), from, to)
}

// keyFiles: files of the base whose sha256 the tester can compare before applying the patch.
func keyFiles(base Manifest) []Entry {
	want := map[string]bool{
		"VERSION.txt": true, "CHANGELOG.md": true, "PartyHardy.exe": true, "PartyHardyEditor.exe": true,
		"PartyHardy.app/Contents/MacOS/PartyHardy": true, "PartyHardy.app/Contents/Info.plist": true,
		"PartyHardyEditor.app/Contents/MacOS/PartyHardyEditor": true,
	}
	var out []Entry
	for _, e := range base.Files {
		if want[e.Path] {
			out = append(out, e)
		}
	}
	return out
}

func leemeText(base, next Manifest, d Diff, changes string) string {
	from, to, plat := base.Version, next.Version, next.Platform
	var b strings.Builder
	w := func(s string, a ...any) { fmt.Fprintf(&b, s+"\n", a...) }
	w("PARTY HARDY - PARCHE %s  v%s -> v%s", platformName(plat), from, to)
	w("==========================================================")
	w("")
	w("[ES] ESTE PARCHE SOLO SIRVE PARA LA VERSIÓN v%s EXACTA (mira VERSION.txt en la carpeta del juego;", from)
	w("     para comprobarlo a fondo, ver VERIFICAR_BASE.txt). Con otra versión NO lo apliques: descarga el juego completo.")
	w("Cómo actualizar:")
	w("  1. Cierra el juego y el editor.")
	w("  2. Por si acaso, copia aparte tu partida y ajustes: assets/progress.json y assets/settings.json")
	w("     (el parche NO los trae ni los toca).")
	if plat == "mac" {
		w("  3. Copia el contenido del parche encima de la carpeta del juego (la que tiene VERSION.txt). Lo más seguro,")
		w("     en Terminal (arrastra el zip y la carpeta a la ventana para escribir sus rutas):")
		w("        unzip -o <este zip> -d <carpeta del juego>")
		w("     ¡OJO! NO arrastres carpetas con Finder eligiendo \"Reemplazar\": borraría la carpeta entera (assets con")
		w("     tu partida incluida). Con Finder, mantén pulsada Opción (⌥) al soltar y elige \"Combinar\".")
	} else {
		w("  3. Descomprime este zip y copia TODO su contenido encima de la carpeta del juego (la que tiene")
		w("     VERSION.txt); cuando Windows pregunte, elige \"Reemplazar los archivos en el destino\".")
	}
	w("  4. Borra de la carpeta del juego lo que aparezca en REMOVIDOS.txt (si está vacío, nada; las rutas usan /,")
	w("     en Windows es lo mismo que \\).")
	w("  5. Abre el juego: abajo a la derecha del menú debe poner v.%s.", to)
	if plat == "mac" {
		w("  En Mac, si dice que la app está dañada o no se puede abrir: clic derecho > Abrir (una vez), o en Terminal:")
		w("     xattr -dr com.apple.quarantine PartyHardy.app PartyHardyEditor.app")
	}
	w("  Los ficheros LEEME_ACTUALIZAR.txt, REMOVIDOS.txt y VERIFICAR_BASE.txt pueden borrarse después.")
	w("")
	w("[EN] THIS PATCH ONLY WORKS ON VERSION v%s EXACTLY (see VERSION.txt in the game folder; VERIFICAR_BASE.txt", from)
	w("     has file hashes to check it). With any other version DO NOT apply it: download the full game.")
	w("How to update:")
	w("  1. Close the game and the editor.")
	w("  2. Just in case, back up assets/progress.json and assets/settings.json (the patch never touches them).")
	if plat == "mac" {
		w("  3. Copy the patch content over the game folder (the one with VERSION.txt). Safest, in Terminal:")
		w("        unzip -o <this zip> -d <game folder>")
		w("     WARNING: do NOT drag folders in Finder choosing \"Replace\": it deletes the whole folder (assets with")
		w("     your save included). In Finder hold Option (⌥) while dropping and choose \"Merge\".")
	} else {
		w("  3. Unzip this file and copy ALL its content over the game folder (the one with VERSION.txt);")
		w("     when Windows asks, choose \"Replace the files in the destination\".")
	}
	w("  4. Delete from the game folder every path listed in REMOVIDOS.txt (if empty, nothing).")
	w("  5. Open the game: the main menu shows v.%s in the bottom right corner.", to)
	if plat == "mac" {
		w("  On Mac, if it says the app is damaged or cannot be opened: right click > Open (once), or in Terminal:")
		w("     xattr -dr com.apple.quarantine PartyHardy.app PartyHardyEditor.app")
	}
	w("  LEEME_ACTUALIZAR.txt, REMOVIDOS.txt and VERIFICAR_BASE.txt can be deleted afterwards.")
	w("")
	w("Contenido / Content: %d nuevos/new, %d cambiados/changed, %d a borrar/to delete.", len(d.Added), len(d.Changed), len(d.Removed))
	w("")
	w("CAMBIOS / CHANGES (CHANGELOG.md, v%s -> v%s)", from, to)
	w("----------------------------------------------------------")
	if changes == "" {
		changes = "(sin entradas en el CHANGELOG / no CHANGELOG entries)"
	}
	b.WriteString(changes + "\n")
	return strings.ReplaceAll(b.String(), "\n", "\r\n") // readable in Notepad too
}

func verificarText(base Manifest) string {
	var b strings.Builder
	w := func(s string, a ...any) { fmt.Fprintf(&b, s+"\n", a...) }
	w("VERIFICAR LA VERSIÓN BASE / CHECK THE BASE VERSION  (v%s, %s)", base.Version, platformName(base.Platform))
	w("")
	w("[ES] Antes de aplicar el parche, VERSION.txt de tu carpeta del juego debe decir: Party Hardy v%s", base.Version)
	w("     Para comprobarlo a fondo compara el SHA-256 de estos ficheros con los de abajo:")
	w("[EN] Before applying the patch, VERSION.txt in your game folder must say: Party Hardy v%s", base.Version)
	w("     For a thorough check compare the SHA-256 of these files with the ones below:")
	if base.Platform == "mac" {
		w("     Mac (Terminal, dentro de la carpeta del juego / inside the game folder):  shasum -a 256 <fichero/file>")
	} else {
		w("     Windows (cmd, dentro de la carpeta del juego / inside the game folder):  certutil -hashfile <fichero/file> SHA256")
	}
	w("")
	for _, e := range keyFiles(base) {
		w("%s  %s", e.SHA256, e.Path)
	}
	return strings.ReplaceAll(b.String(), "\n", "\r\n")
}

// writePatch builds the patch zip: changed and new files with their folders and permissions, plus
// REMOVIDOS.txt, LEEME_ACTUALIZAR.txt and VERIFICAR_BASE.txt. Deterministic for the same inputs.
func writePatch(out string, base Manifest, next *zipContent) (Diff, error) {
	nm := next.Manifest
	d := computeDiff(base, nm)
	var newest time.Time
	for _, f := range next.Reader.File {
		if f.Modified.After(newest) {
			newest = f.Modified
		}
	}
	changes := ""
	if cf, ok := next.Files["CHANGELOG.md"]; ok {
		cb, err := readZipFile(cf)
		if err != nil {
			return d, err
		}
		changes = changelogBetween(string(cb), base.Version, nm.Version)
	}
	removed := strings.Join(d.Removed, "\r\n")
	if removed != "" {
		removed += "\r\n"
	}
	extras := []struct{ name, body string }{
		{"LEEME_ACTUALIZAR.txt", leemeText(base, nm, d, changes)},
		{"REMOVIDOS.txt", removed},
		{"VERIFICAR_BASE.txt", verificarText(base)},
	}
	for _, x := range extras {
		if _, clash := next.Files[x.name]; clash {
			return d, fmt.Errorf("el juego ya tiene un fichero %s: cambia el nombre de los ficheros del parche", x.name)
		}
	}
	tmp := out + ".tmp"
	fh, err := os.Create(tmp)
	if err != nil {
		return d, err
	}
	zw := zip.NewWriter(fh)
	fail := func(e error) (Diff, error) { zw.Close(); fh.Close(); os.Remove(tmp); return d, e }
	files := append(append([]string{}, d.Added...), d.Changed...)
	sort.Strings(files)
	dirSeen := map[string]bool{}
	addDir := func(dir string) error {
		var chain []string
		for x := dir; x != "." && x != "" && !dirSeen[x]; x = parentDir(x) {
			chain = append([]string{x}, chain...)
		}
		for _, x := range chain {
			dirSeen[x] = true
			h := &zip.FileHeader{Name: x + "/", Method: zip.Store, Modified: newest}
			h.SetMode(os.ModeDir | 0o755)
			if _, err := zw.CreateHeader(h); err != nil {
				return err
			}
		}
		return nil
	}
	for _, p := range files {
		if why := forbiddenReason(p); why != "" {
			return fail(fmt.Errorf("%s no puede ir en un parche (%s)", p, why))
		}
		src := next.Files[p]
		if err := addDir(parentDir(p)); err != nil {
			return fail(err)
		}
		h := &zip.FileHeader{Name: p, Method: zip.Deflate, Modified: src.Modified}
		mode := os.FileMode(0o644)
		if src.Mode().Perm()&0o111 != 0 {
			mode = 0o755
		}
		h.SetMode(mode)
		w, err := zw.CreateHeader(h)
		if err != nil {
			return fail(err)
		}
		rc, err := src.Open()
		if err != nil {
			return fail(err)
		}
		_, err = io.Copy(w, rc)
		rc.Close()
		if err != nil {
			return fail(err)
		}
	}
	for _, x := range d.AddedDirs {
		if err := addDir(x); err != nil {
			return fail(err)
		}
	}
	for _, x := range extras {
		h := &zip.FileHeader{Name: x.name, Method: zip.Deflate, Modified: newest}
		h.SetMode(0o644)
		w, err := zw.CreateHeader(h)
		if err != nil {
			return fail(err)
		}
		if _, err := io.Copy(w, bytes.NewReader([]byte(x.body))); err != nil {
			return fail(err)
		}
	}
	if err := zw.Close(); err != nil {
		fh.Close()
		os.Remove(tmp)
		return d, err
	}
	if err := fh.Close(); err != nil {
		os.Remove(tmp)
		return d, err
	}
	return d, os.Rename(tmp, out)
}

func parentDir(p string) string {
	i := strings.LastIndex(p, "/")
	if i < 0 {
		return ""
	}
	return p[:i]
}
