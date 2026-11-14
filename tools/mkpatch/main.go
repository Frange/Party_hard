// mkpatch: manifests of each release and update patches that carry ONLY what changed between two versions.
//
// Manifest of a built version (build.sh runs it after packaging the full zips):
//
//	go -C tools/mkpatch run . manifest -mac <PartyHardy-Mac.zip> -win <PartyHardy-Windows.zip> -dir <releases>
//
// writes releases/manifest-<version>-<mac|win>.json (relative path, size, sha256 and mode of every file of
// the zip; sorted, no dates, no local paths). The version is read from VERSION.txt inside the zip.
//
// Patch from 1.0.0 to the built one (or use ./mkpatch.sh from the project root; owner's policy: one patch from 1.0.0, none between versions):
//
//	go -C tools/mkpatch run . -from 1.0.0 [-to 1.0.1] -mac <zip> -win <zip> -dir <releases> -out <folder>
//
// writes PartyHardy-Parche-<Mac|Windows>-<from>-a-<to>.zip with the new and changed files (same folders as
// the game), REMOVIDOS.txt, LEEME_ACTUALIZAR.txt (ES/EN) and VERIFICAR_BASE.txt. It fails if the manifest of
// the base version does not exist. Independent module: it does not touch the game package.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "mkpatch:", err)
		os.Exit(1)
	}
}

type platformZip struct{ plat, zip string }

func run(args []string) error {
	mode := "patch"
	if len(args) > 0 && args[0] == "manifest" {
		mode, args = "manifest", args[1:]
	}
	fs := flag.NewFlagSet("mkpatch", flag.ContinueOnError)
	from := fs.String("from", "", "versión base (debe existir releases/manifest-<from>-<plat>.json)")
	to := fs.String("to", "", "versión destino (por defecto, la del VERSION.txt del zip nuevo)")
	mac := fs.String("mac", "", "zip completo nuevo de Mac")
	win := fs.String("win", "", "zip completo nuevo de Windows")
	dir := fs.String("dir", "releases", "carpeta de los manifiestos")
	out := fs.String("out", ".", "carpeta donde dejar los parches")
	force := fs.Bool("force", false, "manifest: sobrescribe un manifiesto distinto de la misma versión (sólo si NUNCA se publicó)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	var zips []platformZip
	if *mac != "" {
		zips = append(zips, platformZip{"mac", *mac})
	}
	if *win != "" {
		zips = append(zips, platformZip{"win", *win})
	}
	if len(zips) == 0 {
		return fmt.Errorf("indica al menos -mac o -win con el zip completo")
	}
	if mode == "manifest" {
		for _, z := range zips {
			c, err := openRelease(z.zip, z.plat)
			if err != nil {
				return err
			}
			c.Reader.Close()
			file, err := writeManifest(*dir, c.Manifest, *force)
			if err != nil {
				return err
			}
			fmt.Printf("manifiesto %s: %d ficheros, %d bytes\n", file, len(c.Manifest.Files), c.Manifest.totalSize())
		}
		return nil
	}
	if *from == "" {
		return fmt.Errorf("falta -from <versión base>")
	}
	if _, ok := parseSemver(*from); !ok {
		return fmt.Errorf("-from %q no es una versión X.Y.Z", *from)
	}
	for _, z := range zips {
		mfile := manifestPath(*dir, *from, z.plat)
		base, err := loadManifest(mfile)
		if os.IsNotExist(err) {
			return fmt.Errorf("no existe el manifiesto de la versión base %s para %s (%s): sólo se pueden hacer parches "+
				"desde versiones construidas con manifiesto", *from, platformName(z.plat), mfile)
		}
		if err != nil {
			return err
		}
		if base.Version != *from || base.Platform != z.plat {
			return fmt.Errorf("%s dice versión %s/%s, se esperaba %s/%s", mfile, base.Version, base.Platform, *from, z.plat)
		}
		c, err := openRelease(z.zip, z.plat)
		if err != nil {
			return err
		}
		nv := c.Manifest.Version
		if *to != "" && *to != nv {
			c.Reader.Close()
			return fmt.Errorf("%s es la versión %s, no la %s pedida con -to", z.zip, nv, *to)
		}
		if !semverLess(*from, nv) {
			c.Reader.Close()
			return fmt.Errorf("la versión base %s debe ser anterior a la nueva %s", *from, nv)
		}
		// the destination manifest, when it exists, must describe this same zip
		if dm, err := loadManifest(manifestPath(*dir, nv, z.plat)); err == nil && string(dm.encode()) != string(c.Manifest.encode()) {
			c.Reader.Close()
			return fmt.Errorf("%s no coincide con %s: el zip no es el publicado como v%s", manifestPath(*dir, nv, z.plat), z.zip, nv)
		}
		if err := os.MkdirAll(*out, 0o755); err != nil {
			c.Reader.Close()
			return err
		}
		name := filepath.Join(*out, patchFileName(z.plat, *from, nv))
		d, err := writePatch(name, base, c)
		c.Reader.Close()
		if err != nil {
			return err
		}
		st, _ := os.Stat(name)
		fmt.Printf("parche %s: %d nuevos, %d cambiados, %d a borrar, %d bytes\n", name, len(d.Added), len(d.Changed), len(d.Removed), st.Size())
	}
	return nil
}
