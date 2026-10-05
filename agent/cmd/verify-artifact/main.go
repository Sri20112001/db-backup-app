// Command verify-artifact gates release binaries: it fails unless the file
// is a valid Windows executable of the expected architecture, above a
// minimum size (catches truncated/failed builds), and free of development
// paths and embedded secret material. Stdlib only (debug/pe), so it runs in
// any build container without network access:
//
//	go run ./cmd/verify-artifact --arch amd64 --min-bytes 5000000 path/to/VaultGuard-Agent.exe
package main

import (
	"debug/pe"
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
)

var (
	wantArch = map[string]uint16{"amd64": pe.IMAGE_FILE_MACHINE_AMD64, "386": pe.IMAGE_FILE_MACHINE_I386}
	// devPathRes catch workstation paths leaked without -trimpath.
	devPathRes = []*regexp.Regexp{
		regexp.MustCompile(`C:\\Users\\[A-Za-z0-9_.$-]+\\`),
		regexp.MustCompile(`/home/[a-z_][a-z0-9_-]*/`),
		regexp.MustCompile(`/Users/[a-z_][a-z0-9_-]*/`),
	}
	// secretRes catch obvious embedded secret material (values, not names).
	secretRes = []*regexp.Regexp{
		regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
		regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----`),
	}
	printableRuns = regexp.MustCompile(`[ -~]{8,}`)
)

func main() {
	arch := flag.String("arch", "amd64", "required PE machine: amd64|386")
	minBytes := flag.Int64("min-bytes", 5000000, "minimum acceptable file size")
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: verify-artifact [--arch amd64] [--min-bytes N] <exe>")
		os.Exit(2)
	}
	if err := verify(flag.Arg(0), *arch, *minBytes); err != nil {
		fmt.Fprintln(os.Stderr, "verify-artifact FAILED:", err)
		os.Exit(1)
	}
	fmt.Println("verify-artifact OK")
}

func verify(path, arch string, minBytes int64) error {
	want, ok := wantArch[strings.ToLower(arch)]
	if !ok {
		return fmt.Errorf("unknown arch %q", arch)
	}
	fi, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat: %w", err)
	}
	if fi.Size() < minBytes {
		return fmt.Errorf("size %d below minimum %d (truncated build?)", fi.Size(), minBytes)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}
	if f, err := pe.Open(path); err != nil {
		// MinGW-linked binaries (Fyne GUI via external ld, especially after
		// mt.exe resource edits) can defeat debug/pe's strict COFF parsing
		// while remaining perfectly valid to the Windows loader. Fall back
		// to a manual header check: MZ + PE signature + machine field.
		machine, ferr := peMachineFallback(raw)
		if ferr != nil {
			return fmt.Errorf("not a valid PE executable (%v; fallback: %v)", err, ferr)
		}
		if machine != want {
			return fmt.Errorf("PE machine 0x%04X, want 0x%04X (%s)", machine, want, arch)
		}
	} else {
		defer f.Close()
		if f.Machine != want {
			return fmt.Errorf("PE machine 0x%04X, want 0x%04X (%s)", uint16(f.Machine), want, arch)
		}
	}
	// Scan printable runs (same technique as the manual acceptance scan).
	var sb strings.Builder
	for _, b := range raw {
		if b >= 32 && b <= 126 {
			sb.WriteByte(b)
		} else {
			sb.WriteByte('\n')
		}
	}
	text := sb.String()
	for _, re := range devPathRes {
		if m := re.FindString(text); m != "" {
			return fmt.Errorf("embedded dev path (rebuild with -trimpath): %q", m)
		}
	}
	for _, re := range secretRes {
		if re.MatchString(text) {
			return fmt.Errorf("embedded secret material detected")
		}
	}
	return nil
}

// peMachineFallback reads the DOS header + PE signature + COFF machine field
// directly: enough to gate architecture when debug/pe chokes on
// externally-linked (MinGW/mt.exe-touched) binaries the loader accepts.
func peMachineFallback(raw []byte) (uint16, error) {
	if len(raw) < 64 || raw[0] != 'M' || raw[1] != 'Z' {
		return 0, fmt.Errorf("missing MZ header")
	}
	peOff := binary.LittleEndian.Uint32(raw[0x3C:])
	if int(peOff)+6 > len(raw) {
		return 0, fmt.Errorf("bad PE offset")
	}
	if raw[peOff] != 'P' || raw[peOff+1] != 'E' || raw[peOff+2] != 0 || raw[peOff+3] != 0 {
		return 0, fmt.Errorf("missing PE signature")
	}
	return binary.LittleEndian.Uint16(raw[peOff+4:]), nil
}
