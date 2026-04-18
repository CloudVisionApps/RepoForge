package repoindex

import "fmt"

func DebUploadLogicalPath(tmpDebPath, uploadBaseName string) (string, error) {
	ctrl, err := parseDebControl(tmpDebPath)
	if err != nil {
		return "", fmt.Errorf("read .deb control: %w", err)
	}
	return poolRelativePath(ctrl.get("Package"), uploadBaseName), nil
}
