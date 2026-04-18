package repoindex

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ulikunitz/xz"
)

type debControl map[string]string

func parseDebControl(path string) (debControl, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if err := readArMagic(f); err != nil {
		return nil, err
	}
	var controlTar []byte
	var kind string
	for {
		name, size, err := readArHeader(f)
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				break
			}
			return nil, err
		}
		if k, ok := isControlTarMember(name); ok {
			kind = k
			controlTar, err = readArPayload(f, size)
			if err != nil {
				return nil, err
			}
			break
		}
		if err := skipArPayload(f, size); err != nil {
			return nil, err
		}
	}
	if len(controlTar) == 0 {
		return nil, fmt.Errorf("missing control archive in .deb")
	}
	raw, err := extractControlFile(controlTar, kind)
	if err != nil {
		return nil, err
	}
	return parseControlStanza(raw), nil
}

func extractControlFile(controlTar []byte, kind string) ([]byte, error) {
	if kind == "" {
		switch {
		case len(controlTar) >= 2 && controlTar[0] == 0x1f && controlTar[1] == 0x8b:
			kind = "gz"
		case len(controlTar) >= 6 && bytes.Equal(controlTar[:6], []byte{0xfd, '7', 'z', 'X', 'Z', 0x00}):
			kind = "xz"
		}
	}
	var r io.Reader
	switch kind {
	case "gz":
		gz, err := gzip.NewReader(bytes.NewReader(controlTar))
		if err != nil {
			return nil, err
		}
		defer gz.Close()
		r = gz
	case "xz":
		xr, err := xz.NewReader(bytes.NewReader(controlTar))
		if err != nil {
			return nil, err
		}
		r = xr
	default:
		return nil, fmt.Errorf("unknown control archive type %q", kind)
	}
	tr := tar.NewReader(r)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return nil, fmt.Errorf("control file not found in control.tar")
		}
		if err != nil {
			return nil, err
		}
		if h.Name == "./control" || h.Name == "control" {
			return io.ReadAll(tr)
		}
	}
}

func parseControlStanza(data []byte) debControl {
	m := make(debControl)
	var key string
	var val strings.Builder
	flush := func() {
		if key != "" {
			m[key] = strings.TrimSpace(val.String())
		}
		key = ""
		val.Reset()
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			if key != "" {
				val.WriteString("\n")
				val.WriteString(strings.TrimSpace(line))
			}
			continue
		}
		flush()
		if line == "" {
			continue
		}
		idx := strings.IndexByte(line, ':')
		if idx < 0 {
			continue
		}
		key = strings.TrimSpace(line[:idx])
		val.WriteString(strings.TrimSpace(line[idx+1:]))
	}
	flush()
	return m
}

func (c debControl) get(key string) string {
	return strings.TrimSpace(c[key])
}
