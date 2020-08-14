package utils

import (
	"io/ioutil"
	"path"
	"regexp"
	"strconv"
	"strings"
)

type ProcFSFile struct {
	slurp string
}

func ReadProcFSFile(name string) (*ProcFSFile, error) {
	f := path.Join("/proc", name)

	bytes, err := ioutil.ReadFile(f)
	if err != nil {
		return nil, err
	}

	p := &ProcFSFile{
		slurp: string(bytes),
	}
	return p, nil
}

func (p *ProcFSFile) GetNumberValue(prefix string) (uint64, error) {
	expr := regexp.MustCompile(prefix + `\s+(\d+)`)
	m := expr.FindStringSubmatch(p.slurp)
	if len(m) == 2 {
		v, err := strconv.ParseUint(m[1], 10, 64)
		if err != nil {
			return 0, err
		}
		return v, nil
	}
	return 0, nil
}

func (p *ProcFSFile) GetStringValue(prefix string) (string, error) {
	expr := regexp.MustCompile(prefix + `\s+(.+)`)
	m := expr.FindStringSubmatch(p.slurp)
	if len(m) == 2 {
		return strings.TrimSpace(m[1]), nil
	}
	return "", nil
}

func (p *ProcFSFile) GetAsString() string {
	return p.slurp
}

func (p *ProcFSFile) GetAsLines() []string {
	return strings.Split(p.slurp, "\n")
}
