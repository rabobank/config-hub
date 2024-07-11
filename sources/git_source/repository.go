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
	"sync"

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

type Repository struct {
	shallow bool
	base    string
	pull    []string
	lock    sync.Mutex
}

func Git(config *domain.GitConfig, baseDir string) (*Repository, error) {
	var repoPath string
	if strings.HasPrefix(config.Uri, "git@") {
		repoPath = config.Uri[strings.Index(config.Uri, ":")+1:]
	} else {
		// let's ignore the error for now
		gitUrl, _ := url.Parse(config.Uri)
		repoPath = strings.ReplaceAll(gitUrl.Path, " ", "%20")
	}

	repository := &Repository{
		shallow: !config.DeepClone,
		base:    baseDir,
	}

	if repository.shallow {
		repository.pull = []string{"pull", "--depth=1"}
	} else {
		repository.pull = []string{"pull"}
	}

	repository.lock.Lock()
	defer repository.lock.Unlock()

	if output, e := repository.exec([]string{"init"}); e != nil {
		l.Error(output)
		return nil, e
	}

	if output, e := repository.exec([]string{"config", "--add", "credential.helper", fmt.Sprintf(CredentialHelperCommand, path.Join(cfg.BaseDir, path.Base(os.Args[0])), repoPath)}); e != nil {
		l.Error(output)
		return nil, e
	}

	if output, e := repository.exec([]string{"config", "--add", "advice.detachedHead", "false"}); e != nil {
		l.Error(output)
		return nil, e
	}

	if output, e := repository.exec([]string{"remote", "add", "origin", config.Uri}); e != nil {
		l.Error(output)
		return nil, e
	}

	if output, e := repository.exec([]string{"config", "pull.rebase", "true"}); e != nil {
		l.Error(output)
		return nil, e
	}

	return repository, nil
}

func (r *Repository) Fetch(label string) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	fetch := []string{"fetch"}
	if r.shallow {
		fetch = append(fetch, "--depth=1")
		if len(label) > 0 {
			fetch = append(fetch, "origin", label)
		}
	}
	if output, e := r.exec(fetch); e != nil {
		l.Error(output)
		return e
	} else if l.Level() >= log.DEBUG {
		l.Debug(output)
	}

	return nil
}

func (r *Repository) Refresh(label string) error {
	if e := r.Fetch(label); e != nil {
		return e
	}

	r.lock.Lock()
	defer r.lock.Unlock()

	if output, e := r.exec([]string{"checkout", label}); e != nil {
		l.Error(output)
		return e
	} else if l.Level() >= log.DEBUG {
		l.Debug(output)
	}

	if output, e := r.exec(r.pull); e != nil {
		l.Error(output)
		return e
	} else if l.Level() >= log.DEBUG {
		l.Debug(output)
	}

	return nil
}

func (r *Repository) Branches(remote bool) ([]Branch, error) {
	if e := r.Fetch(""); e != nil {
		l.Error("Listing Branches failed on fetch:", e)
		return nil, e
	}

	parameters := localBranchParameters
	if remote {
		l.Debug("Listing remote branches")
		parameters = remoteBranchParameters
	} else {
		l.Debug("Listing local branches")
	}
	if output, e := r.exec(parameters); e != nil {
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

func (r *Repository) Exec(parameters ...string) (*bytes.Buffer, error) {
	r.lock.Lock()
	defer r.lock.Unlock()
	return r.exec(parameters)
}

func (r *Repository) exec(parameters []string) (*bytes.Buffer, error) {
	cmd := exec.Command("git", parameters...)
	cmd.Dir = r.base
	cmd.Env = os.Environ()
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if e := cmd.Run(); e != nil {
		return &buf, e
	}
	return &buf, nil
}
