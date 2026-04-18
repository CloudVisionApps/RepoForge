package repoindex

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

const arGlobalMagic = "!<arch>\n"

func readArMagic(r io.Reader) error {
	buf := make([]byte, len(arGlobalMagic))
	if _, err := io.ReadFull(r, buf); err != nil {
		return err
	}
	if string(buf) != arGlobalMagic {
		return fmt.Errorf("not an ar archive")
	}
	return nil
}

func readArHeader(r io.Reader) (name string, size int64, err error) {
	hdr := make([]byte, 60)
	if _, err := io.ReadFull(r, hdr); err != nil {
		return "", 0, err
	}
	if hdr[58] != '`' || hdr[59] != '\n' {
		return "", 0, fmt.Errorf("invalid ar header terminator")
	}
	name = strings.TrimSpace(string(hdr[0:16]))
	sizeStr := strings.TrimSpace(string(hdr[48:58]))
	size, err = strconv.ParseInt(sizeStr, 10, 64)
	if err != nil || size < 0 {
		return "", 0, fmt.Errorf("invalid ar member size %q", sizeStr)
	}
	return name, size, nil
}

func skipArPayload(r io.Reader, size int64) error {
	if _, err := io.CopyN(io.Discard, r, size); err != nil {
		return err
	}
	if size&1 == 1 {
		_, err := io.CopyN(io.Discard, r, 1)
		return err
	}
	return nil
}

func readArPayload(r io.Reader, size int64) ([]byte, error) {
	out := make([]byte, size)
	if _, err := io.ReadFull(r, out); err != nil {
		return nil, err
	}
	if size&1 == 1 {
		if _, err := io.CopyN(io.Discard, r, 1); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func isControlTarMember(name string) (kind string, ok bool) {
	name = strings.TrimSpace(name)
	switch {
	case name == "control.tar.gz" || strings.HasPrefix(name, "control.tar.gz/"):
		return "gz", true
	case name == "control.tar.xz" || strings.HasPrefix(name, "control.tar.xz/"):
		return "xz", true
	case strings.HasPrefix(name, "control.tar.zst"):
		return "", false
	default:
		return "", false
	}
}
