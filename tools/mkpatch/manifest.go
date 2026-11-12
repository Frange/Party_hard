package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// Entry: one file of a full release zip, relative to the game folder (the folder that holds VERSION.txt).
type Entry struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
	Mode   string `json:"mode"` // unix permissions in octal ("0755" for executables)
}

// Manifest: exactly what the full zip of one version and platform contains. Sorted and deterministic:
// no dates, no local paths, no personal data.
type Manifest struct {
	Game      string   `json:"game"`
	Version   string   `json:"version"`
	Platform  string   `json:"platform"` // "mac" or "win"
	ZipPrefix string   `json:"zip_prefix"`
	Files     []Entry  `json:"files"`
	EmptyDirs []string `json:"empty_dirs"`
}

// zipContent: a full release zip read into memory-light form (manifest + the zip entries to copy from).
type zipContent struct {
	Manifest Manifest
	Files    map[string]*zip.File // by relative path
	Reader   *zip.ReadCloser
}

// forbiddenReason: why a path must never be inside a release or a patch ("" when it is fine).
// The full zips never carry personal data, imports, backups or intros music: build.sh removes them.
func forbiddenReason(rel string) string {
	base := path.Base(rel)
	parts := strings.Split(rel, "/")
	for _, p := range parts {
		switch {
		case p == "imports", p == "AI_with_Claude", strings.HasPrefix(p, "assets_backup"), p == "__MACOSX":
			return "carpeta excluida " + p
		}
	}
	switch {
	case strings.HasPrefix(base, "progress") && strings.Contains(base, ".json"):
		return "partida del usuario"
	case strings.HasPrefix(base, "settings.json"), strings.HasPrefix(base, "editor_prefs.json"):
		return "ajustes del usuario"
	case base == ".DS_Store", strings.HasPrefix(base, "._"), base == ".held_objects_fixed":
		return "fichero de sistema"
	case len(parts) == 2 && parts[0] == "intros" && strings.HasSuffix(strings.ToLower(base), ".mp3"):
		return "música de intros (no se distribuye)"
	}
	return ""
}

// readVersionTxt: "Party Hardy v1.0.0" -> "1.0.0".
func readVersionTxt(b []byte) (string, error) {
	s := strings.TrimSpace(string(b))
	i := strings.LastIndex(s, " v")
	if !strings.HasPrefix(s, "Party Hardy v") || i < 0 {
		return "", fmt.Errorf("VERSION.txt no tiene el formato 'Party Hardy vX.Y.Z': %q", s)
	}
	v := s[i+2:]
	if _, ok := parseSemver(v); !ok {
		return "", fmt.Errorf("versión no válida en VERSION.txt: %q", v)
	}
	return v, nil
}

// openRelease reads a full release zip: finds the game folder (where VERSION.txt is), hashes every file and
// refuses zips with personal data or files outside the game folder.
func openRelease(zipPath, platform string) (*zipContent, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, err
	}
	fail := func(e error) (*zipContent, error) { r.Close(); return nil, e }
	prefix := "-"
	for _, f := range r.File {
		if path.Base(f.Name) == "VERSION.txt" && !f.FileInfo().IsDir() {
			p := strings.TrimSuffix(f.Name, "VERSION.txt")
			if prefix == "-" || len(p) < len(prefix) {
				prefix = p
			}
		}
	}
	if prefix == "-" {
		return fail(fmt.Errorf("%s: no contiene VERSION.txt", zipPath))
	}
	c := &zipContent{Files: map[string]*zip.File{}, Reader: r}
	m := &c.Manifest
	m.Game, m.Platform, m.ZipPrefix = "Party Hardy", platform, prefix
	dirs := map[string]bool{}
	parentOfFile := map[string]bool{}
	for _, f := range r.File {
		name := strings.TrimSuffix(f.Name, "/")
		if name+"/" == prefix || name == "" {
			continue
		}
		if !strings.HasPrefix(f.Name, prefix) {
			return fail(fmt.Errorf("%s: %q está fuera de la carpeta del juego %q", zipPath, f.Name, prefix))
		}
		rel := strings.TrimPrefix(name, prefix)
		if strings.Contains(rel, "..") || strings.HasPrefix(rel, "/") || strings.Contains(rel, "\\") {
			return fail(fmt.Errorf("%s: ruta no válida %q", zipPath, f.Name))
		}
		if why := forbiddenReason(rel); why != "" {
			return fail(fmt.Errorf("%s: contiene %q (%s): el zip completo no debe llevarlo", zipPath, rel, why))
		}
		if f.FileInfo().IsDir() {
			dirs[rel] = true
			continue
		}
		if f.Mode()&os.ModeSymlink != 0 {
			return fail(fmt.Errorf("%s: enlace simbólico no soportado %q", zipPath, f.Name))
		}
		if _, dup := c.Files[rel]; dup {
			return fail(fmt.Errorf("%s: entrada repetida %q", zipPath, rel))
		}
		rc, err := f.Open()
		if err != nil {
			return fail(err)
		}
		h := sha256.New()
		n, err := io.Copy(h, rc)
		rc.Close()
		if err != nil {
			return fail(fmt.Errorf("%s: %s: %v", zipPath, f.Name, err))
		}
		c.Files[rel] = f
		m.Files = append(m.Files, Entry{Path: rel, Size: n, SHA256: hex.EncodeToString(h.Sum(nil)), Mode: modeString(f.Mode())})
		for d := path.Dir(rel); d != "."; d = path.Dir(d) {
			parentOfFile[d] = true
		}
	}
	for d := range dirs {
		hasChild := parentOfFile[d]
		for o := range dirs {
			if strings.HasPrefix(o, d+"/") {
				hasChild = true
			}
		}
		if !hasChild {
			m.EmptyDirs = append(m.EmptyDirs, d)
		}
	}
	vf, ok := c.Files["VERSION.txt"]
	if !ok {
		return fail(fmt.Errorf("%s: no hay VERSION.txt en la carpeta del juego", zipPath))
	}
	vb, err := readZipFile(vf)
	if err != nil {
		return fail(err)
	}
	if m.Version, err = readVersionTxt(vb); err != nil {
		return fail(fmt.Errorf("%s: %v", zipPath, err))
	}
	m.normalize()
	return c, nil
}

func modeString(m os.FileMode) string {
	if m.Perm()&0o111 != 0 {
		return "0755"
	}
	return "0644"
}

func readZipFile(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

func (m *Manifest) normalize() {
	sort.Slice(m.Files, func(i, j int) bool { return m.Files[i].Path < m.Files[j].Path })
	sort.Strings(m.EmptyDirs)
	if m.Files == nil {
		m.Files = []Entry{}
	}
	if m.EmptyDirs == nil {
		m.EmptyDirs = []string{}
	}
}

// encode: deterministic JSON (sorted, one file per line, trailing newline).
func (m Manifest) encode() []byte {
	m.normalize()
	var b bytes.Buffer
	head, _ := json.Marshal(struct {
		Game      string `json:"game"`
		Version   string `json:"version"`
		Platform  string `json:"platform"`
		ZipPrefix string `json:"zip_prefix"`
		FileCount int    `json:"file_count"`
		TotalSize int64  `json:"total_size"`
	}{m.Game, m.Version, m.Platform, m.ZipPrefix, len(m.Files), m.totalSize()})
	b.Write(head[:len(head)-1])
	b.WriteString(",\n\"files\": [\n")
	for i, e := range m.Files {
		line, _ := json.Marshal(e)
		b.Write(line)
		if i < len(m.Files)-1 {
			b.WriteByte(',')
		}
		b.WriteByte('\n')
	}
	b.WriteString("],\n\"empty_dirs\": ")
	d, _ := json.Marshal(m.EmptyDirs)
	b.Write(d)
	b.WriteString("\n}\n")
	return b.Bytes()
}

func (m Manifest) totalSize() int64 {
	var t int64
	for _, e := range m.Files {
		t += e.Size
	}
	return t
}

func manifestPath(dir, version, platform string) string {
	return filepath.Join(dir, fmt.Sprintf("manifest-%s-%s.json", version, platform))
}

func loadManifest(file string) (Manifest, error) {
	var m Manifest
	b, err := os.ReadFile(file)
	if err != nil {
		return m, err
	}
	if err := json.Unmarshal(b, &m); err != nil {
		return m, fmt.Errorf("%s: %v", file, err)
	}
	m.normalize()
	return m, nil
}

// writeManifest writes the manifest of a version; an existing different manifest of the same version is
// never overwritten without force (it is the base of the patches the testers apply).
func writeManifest(dir string, m Manifest, force bool) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	file := manifestPath(dir, m.Version, m.Platform)
	data := m.encode()
	if old, err := os.ReadFile(file); err == nil && !bytes.Equal(old, data) && !force {
		return file, fmt.Errorf("%s ya existe y es distinto: la versión %s ya se publicó con otro contenido. "+
			"Sube la versión, o usa -force si esa versión NUNCA llegó a los testers", file, m.Version)
	}
	return file, os.WriteFile(file, data, 0o644)
}
