package git_source

import (
	"bufio"
	"bytes"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/gomatbase/go-log"
	"github.com/rabobank/config-hub/cfg"
	"github.com/rabobank/config-hub/domain"
)

const (
	Remote                  = true
	Local                   = false
	CredentialHelperCommand = "%s credentials %s"
)

var (
	localBranchParameters  = []string{"branch", "--format", "%(objectname)%(authordate:iso)%(refname:short)"}
	remoteBranchParameters = []string{"branch", "--format", "%(objectname)%(authordate:iso)%(refname:short)", "--remote"}
)

type Branch struct {
	Name     string
	CommitId string
	Date     string
}

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

func listBranches(baseDir string, remote bool) ([]Branch, error) {
	if output, e := git(baseDir, "fetch"); e != nil {
		l.Error(output)
	} else if l.Level() >= log.DEBUG {
		l.Debug(output)
	}

	parameters := localBranchParameters
	if remote {
		l.Debug("Listing remote branches")
		parameters = remoteBranchParameters
	} else {
		l.Debug("Listing local branches")
	}
	if output, e := git(baseDir, parameters...); e != nil {
		l.Error(output)
		return nil, e
	} else {
		var branches []Branch
		scanner := bufio.NewScanner(output)
		for scanner.Scan() {
			branch := scanner.Text()
			branches = append(branches, Branch{
				Name:     branch[65:],
				Date:     branch[40:65],
				CommitId: branch[:40],
			})
		}
		return branches, scanner.Err()
	}
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
