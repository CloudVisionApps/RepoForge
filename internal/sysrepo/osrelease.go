package sysrepo

import (
	"bufio"
	"os"
	"strings"
)

func readOSReleaseID() (id string, idLike string, err error) {
	f, err := os.Open("/etc/os-release")
	if err != nil {
		return "", "", err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "ID=") {
			id = unquote(strings.TrimPrefix(line, "ID="))
		}
		if strings.HasPrefix(line, "ID_LIKE=") {
			idLike = unquote(strings.TrimPrefix(line, "ID_LIKE="))
		}
	}
	return id, idLike, sc.Err()
}

func unquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return strings.Trim(s, `"`)
	}
	return s
}

func idMatches(id, idLike string, want ...string) bool {
	ids := strings.Fields(strings.ReplaceAll(strings.ReplaceAll(idLike, ",", " "), ":", " "))
	ids = append([]string{id}, ids...)
	for _, w := range want {
		for _, x := range ids {
			if strings.EqualFold(strings.TrimSpace(x), w) {
				return true
			}
		}
	}
	return false
}
