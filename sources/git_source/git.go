package git_source

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strings"

	"config-hub/cfg"
	"config-hub/domain"
	"github.com/gomatbase/go-log"
)

const (
	CredentialHelperCommand = "%s credentials %s"
)

func initializeGitRepository(config *domain.GitConfig, baseDir string) error {
	var repoPath string
	if strings.HasPrefix(config.Uri, "git@") {
		repoPath = config.Uri[strings.Index(config.Uri, ":")+1:]
	} else {
		// let's ignore the error for now
		gitUrl, _ := url.Parse(config.Uri)
		repoPath = strings.ReplaceAll(gitUrl.Path, " ", "%20")
	}

	if output, e := git(baseDir, "init"); e != nil {
		l.Error(output)
		return e
	}

	if output, e := git(baseDir, "config", "--add", "credential.helper", fmt.Sprintf(CredentialHelperCommand, path.Join(cfg.BaseDir, path.Base(os.Args[0])), repoPath)); e != nil {
		l.Error(output)
		return e
	}

	if output, e := git(baseDir, "remote", "add", "origin", config.Uri); e != nil {
		l.Error(output)
		return e
	}

	return nil
}

func refresh(baseDir string, label string) error {
	if output, e := git(baseDir, "fetch"); e != nil {
		l.Error(output)
		return e
	} else if l.Level() >= log.DEBUG {
		l.Debug(output)
	}

	if output, e := git(baseDir, "checkout", label); e != nil {
		l.Error(output)
		return e
	}

	if output, e := git(baseDir, "pull"); e != nil {
		l.Error(output)
		return e
	}

	return nil
}

func git(workingDir string, parameters ...string) (*bytes.Buffer, error) {
	cmd := exec.Command("git", parameters...)
	cmd.Dir = workingDir
	cmd.Env = os.Environ()
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if e := cmd.Run(); e != nil {
		return &buf, e
	}
	return &buf, nil
}
