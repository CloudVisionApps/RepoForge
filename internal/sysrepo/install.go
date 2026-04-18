package sysrepo

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

const installTimeout = 20 * time.Minute

// InstallRepoTooling installs distro packages needed for RPM metadata (createrepo_c)
// and common Debian archive tooling (dpkg-dev). repoforge generates DEB indexes in-process,
// but dpkg-dev is still useful on hosts for inspecting or building .deb files.
//
// Must run as root (effective UID 0). Uses only fixed argument lists (no shell).
func InstallRepoTooling(ctx context.Context) (distro string, log string, err error) {
	if os.Geteuid() != 0 {
		return "", "", fmt.Errorf("must run as root (euid=0) to install packages")
	}
	id, idLike, err := readOSReleaseID()
	if err != nil {
		return "", "", fmt.Errorf("read /etc/os-release: %w", err)
	}
	distro = fmt.Sprintf("ID=%s ID_LIKE=%s", id, idLike)

	ctx, cancel := context.WithTimeout(ctx, installTimeout)
	defer cancel()

	var b strings.Builder
	appendRun := func(title string, cmd *exec.Cmd) error {
		fmt.Fprintf(&b, "== %s ==\n$ %s\n", title, strings.Join(cmd.Args, " "))
		out, err := cmd.CombinedOutput()
		b.Write(out)
		if !bytes.HasSuffix(out, []byte("\n")) {
			b.WriteByte('\n')
		}
		if err != nil {
			return fmt.Errorf("%s: %w\n%s", title, err, string(out))
		}
		return nil
	}

	switch {
	case idMatches(id, idLike, "debian", "ubuntu", "linuxmint", "pop"):
		if err := appendRun("apt-get update", exec.CommandContext(ctx, "apt-get", "update", "-qq")); err != nil {
			return distro, b.String(), err
		}
		cmd := exec.CommandContext(ctx, "apt-get", "install", "-yqq",
			"createrepo-c", "rpm", "dpkg-dev",
		)
		cmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
		if err := appendRun("apt-get install", cmd); err != nil {
			return distro, b.String(), err
		}
		return distro, b.String(), nil

	case idMatches(id, idLike, "fedora", "rhel", "centos", "rocky", "almalinux", "ol", "oracle", "amzn", "virtuozzo"):
		if path, e := exec.LookPath("dnf"); e == nil && path != "" {
			cmd := exec.CommandContext(ctx, "dnf", "install", "-y", "createrepo_c", "rpm")
			if err := appendRun("dnf install", cmd); err != nil {
				return distro, b.String(), err
			}
			return distro, b.String(), nil
		}
		if path, e := exec.LookPath("yum"); e == nil && path != "" {
			cmd := exec.CommandContext(ctx, "yum", "install", "-y", "createrepo_c", "rpm")
			if err := appendRun("yum install (createrepo_c)", cmd); err != nil {
				cmd2 := exec.CommandContext(ctx, "yum", "install", "-y", "createrepo", "rpm")
				if err2 := appendRun("yum install (createrepo)", cmd2); err2 != nil {
					return distro, b.String(), fmt.Errorf("yum: createrepo_c failed (%v), createrepo failed (%v)", err, err2)
				}
			}
			return distro, b.String(), nil
		}
		return distro, b.String(), fmt.Errorf("dnf or yum not found in PATH")

	case idMatches(id, idLike, "arch", "manjaro", "endeavouros"):
		cmd := exec.CommandContext(ctx, "pacman", "-Sy", "--noconfirm", "createrepo", "rpm", "dpkg")
		if err := appendRun("pacman install", cmd); err != nil {
			return distro, b.String(), err
		}
		return distro, b.String(), nil

	case idMatches(id, idLike, "opensuse-leap", "opensuse-tumbleweed", "sles"):
		if path, e := exec.LookPath("zypper"); e == nil && path != "" {
			cmd := exec.CommandContext(ctx, "zypper", "--non-interactive", "install", "-y",
				"createrepo_c", "rpm",
			)
			if err := appendRun("zypper install", cmd); err != nil {
				return distro, b.String(), err
			}
			return distro, b.String(), nil
		}
		return distro, b.String(), fmt.Errorf("zypper not found in PATH")

	default:
		return distro, "", fmt.Errorf("unsupported distro ID=%q: install createrepo_c (RPM) and dpkg-dev (optional DEB tooling) manually", id)
	}
}
