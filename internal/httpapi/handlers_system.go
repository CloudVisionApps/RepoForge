package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
	"os"

	"repoforge/internal/sysrepo"
)

type installToolingBody struct {
	Confirm bool `json:"confirm"`
}

func (a *API) postInstallRepoTooling(w http.ResponseWriter, r *http.Request) {
	if a.cfg.BearerToken == "" {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"error":  "REPOFORGE_TOKEN must be set for this endpoint",
			"detail": "Installing system packages is disabled until an API token is configured.",
		})
		return
	}
	var body installToolingBody
	if err := json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&body); err != nil || !body.Confirm {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error":  "invalid body",
			"detail": `Send JSON {"confirm": true} to acknowledge privileged package installation.`,
		})
		return
	}
	if os.Geteuid() != 0 {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"error":  "must run as root",
			"detail": "The repoforge process effective UID must be 0 so the package manager can install createrepo_c and related packages.",
		})
		return
	}
	distro, log, err := sysrepo.InstallRepoTooling(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"ok":     false,
			"error":  err.Error(),
			"detail": err.Error(),
			"distro": distro,
			"log":    log,
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":     true,
		"distro": distro,
		"log":    log,
	})
}
