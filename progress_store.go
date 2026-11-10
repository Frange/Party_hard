package main

import (
	"bytes"
	"compress/zlib"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"os"
	"time"
)

// ---------------------------------------------------------------------------
// How progress.json is stored on disk (it used to be plain, hand-editable JSON).
//
//   file = "PHSAVE1\n" + base64( nonce[16] | AES-CTR(zlib(json)) | HMAC-SHA256[32] )
//
// The keys are derived from a fixed salt of the program, and the HMAC covers the header, the nonce and the
// ciphertext: a file changed by hand (even one byte) fails the check and is treated as corrupt (the last good copy,
// progress.json.bak, is used instead; if that fails too, a new game). HONESTY: the code is public, so the salt is
// public too; this only stops casual editing (it is a deterrent, not real security).
//
// Compatibility: a plain JSON save from before (starts with '{') is still read, and is written in the new format on
// the next normal save. To stop that door from being an easy way back to hand edits, a plain JSON file is only
// accepted while there is no signed copy (.bak) next to it.
// ---------------------------------------------------------------------------

const progressMagic = "PHSAVE1\n"

var progressSalt = []byte("PartyHardy/progress/v1/9f3c1e7a-confetti-and-cake")

func progressKey(purpose string) []byte {
	h := sha256.Sum256(append(append([]byte(nil), progressSalt...), purpose...))
	return h[:]
}

var errProgressBad = errors.New("progress: bad or tampered file")

// encodeProgress turns the progress into the bytes written to disk.
func encodeProgress(p Progress) ([]byte, error) {
	js, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	var zb bytes.Buffer
	zw := zlib.NewWriter(&zb)
	zw.Write(js)
	zw.Close()
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	blk, _ := aes.NewCipher(progressKey("enc"))
	ct := make([]byte, zb.Len())
	cipher.NewCTR(blk, nonce).XORKeyStream(ct, zb.Bytes())
	mac := hmac.New(sha256.New, progressKey("mac"))
	mac.Write([]byte(progressMagic))
	mac.Write(nonce)
	mac.Write(ct)
	raw := append(append(nonce, ct...), mac.Sum(nil)...)
	out := []byte(progressMagic)
	out = append(out, base64.StdEncoding.EncodeToString(raw)...)
	return append(out, '\n'), nil
}

// decodeProgress reads a signed save. legacyOK lets a plain JSON save (the old format) through.
// It returns signed=true when the data came from the signed format.
func decodeProgress(data []byte, legacyOK bool) (p Progress, signed bool, err error) {
	if bytes.HasPrefix(data, []byte(progressMagic)) {
		raw, e := base64.StdEncoding.Strict().DecodeString(string(bytes.TrimSpace(data[len(progressMagic):])))
		if e != nil || len(raw) < 16+32 {
			return p, false, errProgressBad
		}
		nonce, ct, sum := raw[:16], raw[16:len(raw)-32], raw[len(raw)-32:]
		mac := hmac.New(sha256.New, progressKey("mac"))
		mac.Write([]byte(progressMagic))
		mac.Write(nonce)
		mac.Write(ct)
		if !hmac.Equal(sum, mac.Sum(nil)) {
			return p, false, errProgressBad
		}
		blk, _ := aes.NewCipher(progressKey("enc"))
		zb := make([]byte, len(ct))
		cipher.NewCTR(blk, nonce).XORKeyStream(zb, ct)
		zr, e := zlib.NewReader(bytes.NewReader(zb))
		if e != nil {
			return p, false, errProgressBad
		}
		js, e := io.ReadAll(io.LimitReader(zr, 16<<20))
		if e != nil {
			return p, false, errProgressBad
		}
		if json.Unmarshal(js, &p) != nil {
			return Progress{}, false, errProgressBad
		}
		return p, true, nil
	}
	t := bytes.TrimSpace(data)
	if legacyOK && len(t) > 0 && t[0] == '{' {
		if json.Unmarshal(t, &p) != nil {
			return Progress{}, false, errProgressBad
		}
		return p, false, nil
	}
	return p, false, errProgressBad
}

// readProgressFile loads the save at path, falling back to path+".bak". found=false: nothing usable (a new game).
func readProgressFile(path string) (p Progress, found bool) {
	bak, bakErr := os.ReadFile(path + ".bak")
	bakSigned := false
	var bakP Progress
	bakOK := false
	if bakErr == nil {
		if bp, s, err := decodeProgress(bak, true); err == nil {
			bakP, bakSigned, bakOK = bp, s, true
		}
	}
	if data, err := os.ReadFile(path); err == nil {
		if mp, signed, err := decodeProgress(data, !bakSigned); err == nil {
			if !signed && mp.Started == 0 { // an old plain save: when was it begun (the file's age)
				mp.Started = fileStartedUnix(path)
			}
			return mp, true
		}
	}
	if bakOK {
		if !bakSigned && bakP.Started == 0 {
			bakP.Started = fileStartedUnix(path + ".bak")
		}
		return bakP, true
	}
	return Progress{}, false
}

// writeProgressFile writes p atomically (tmp + rename), keeping the previous good file as path+".bak".
func writeProgressFile(path string, p Progress) error {
	data, err := encodeProgress(p)
	if err != nil {
		return err
	}
	if old, err := os.ReadFile(path); err == nil {
		if _, _, err := decodeProgress(old, true); err == nil {
			tmpB := path + ".bak.tmp"
			if os.WriteFile(tmpB, old, 0o644) == nil {
				os.Rename(tmpB, path+".bak")
			}
		}
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}

// fileStartedUnix: the oldest reasonable date of a file (its birth time where the system keeps it, else its
// modification time); now if there is none.
func fileStartedUnix(path string) int64 {
	st, err := os.Stat(path)
	if err != nil {
		return time.Now().Unix()
	}
	t := st.ModTime().Unix()
	if b := fileBirthUnix(st); b > 0 && b < t {
		t = b
	}
	return t
}
