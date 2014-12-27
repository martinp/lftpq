package site

import (
	"fmt"
	"strings"
	"time"
)

type Dir struct {
	Created   time.Time
	Name      string
	IsSymlink bool
}

func ParseDir(s string) (Dir, error) {
	words := strings.SplitN(s, " ", 5)
	if len(words) != 5 {
		return Dir{}, fmt.Errorf("expected 5 words, found %d", len(words))
	}
	t := strings.Join(words[:4], " ")
	created, err := time.Parse("2006-01-02 15:04:05 -0700 MST", t)
	if err != nil {
		return Dir{}, err
	}
	name := words[4]
	isSymlink := strings.HasSuffix(name, "@")
	name = strings.TrimRight(name, "@/")
	return Dir{
		Name:      name,
		Created:   created,
		IsSymlink: isSymlink,
	}, nil
}

func (d *Dir) CreatedAfter(age time.Duration) bool {
	return d.Created.After(time.Now().Add(-age))
}

func (d *Dir) MatchAny(ss []string) bool {
	for _, s := range ss {
		if d.Match(s) {
			return true
		}
	}
	return false
}

func (d *Dir) Match(s string) bool {
	return strings.HasPrefix(d.Name, s)
}