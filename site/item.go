package site

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/martinp/lftpq/lftp"
	"github.com/martinp/lftpq/parser"
)

type Items []Item

func (s Items) Len() int {
	return len(s)
}

func (s Items) Less(i, j int) bool {
	return s[i].Dir.Path < s[j].Dir.Path
}

func (s Items) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

type Item struct {
	lftp.Dir
	LocalDir string
	Transfer bool
	Reason   string
	Media    parser.Media
	*Queue   `json:"-"`
}

func (i *Item) String() string {
	return fmt.Sprintf("Path=%q LocalDir=%q Transfer=%t Reason=%q", i.Path, i.LocalDir, i.Transfer, i.Reason)
}

func (i *Item) DstDir() string {
	// When LocalDir has a trailing slash, the actual destination dir will be a directory inside LocalDir (same
	// behaviour as rsync)
	if strings.HasSuffix(i.LocalDir, string(os.PathSeparator)) {
		return filepath.Join(i.LocalDir, i.Dir.Base())
	}
	return i.LocalDir
}

func (i *Item) IsDstDirEmpty() bool {
	dirs, _ := ioutil.ReadDir(i.DstDir())
	return len(dirs) == 0
}

func (i *Item) Weight() int {
	for _i, p := range i.Queue.priorities {
		if i.Dir.Match(p) {
			return len(i.Queue.priorities) - _i
		}
	}
	return 0
}

func (i *Item) Accept(reason string) {
	i.Transfer = true
	i.Reason = reason
}

func (i *Item) Reject(reason string) {
	i.Transfer = false
	i.Reason = reason
}

func (i *Item) parseLocalDir() (string, error) {
	if i.Queue.localDir == nil {
		return "", fmt.Errorf("template is not set")
	}
	var b bytes.Buffer
	if err := i.Queue.localDir.Execute(&b, i.Media); err != nil {
		return "", err
	}
	return b.String(), nil
}

func (i *Item) setMetadata() {
	m, err := i.Queue.parser(i.Dir.Base())
	if err != nil {
		i.Reject(err.Error())
		return
	}
	i.Media = m

	d, err := i.parseLocalDir()
	if err != nil {
		i.Reject(err.Error())
		return
	}
	i.LocalDir = d
}

func newItem(q *Queue, d lftp.Dir) Item {
	item := Item{Queue: q, Dir: d, Reason: "no match"}
	item.setMetadata()
	return item
}
