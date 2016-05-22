package queue

import (
	"os"
	"regexp"
	"sort"
	"testing"
	"text/template"

	"github.com/martinp/lftpq/lftp"
	"github.com/martinp/lftpq/parser"
)

func newTestItem(q *Queue, dir lftp.File) Item {
	item, _ := newItem(q, dir)
	return item
}

func TestNewItemShow(t *testing.T) {
	tmpl, err := parseTemplate(`/tmp/{{ .Name }}/S{{ .Season | Sprintf "%02d" }}/`)
	if err != nil {
		t.Fatal(err)
	}
	s := Site{
		localDir: tmpl,
		parser:   parser.Show,
	}
	d := lftp.File{Path: "/foo/The.Wire.S03E01"}
	q := Queue{Site: s}
	item := newTestItem(&q, d)
	if expected := "/tmp/The.Wire/S03/"; item.LocalDir != expected {
		t.Fatalf("Expected %q, got %q", expected, item.LocalDir)
	}
}

func TestNewItemMovie(t *testing.T) {
	tmpl := template.Must(template.New("").Parse(
		"/tmp/{{ .Year }}/{{ .Name }}/"))
	s := Site{
		localDir: tmpl,
		parser:   parser.Movie,
	}
	d := lftp.File{Path: "/foo/Apocalypse.Now.1979"}
	q := Queue{Site: s}
	item := newTestItem(&q, d)
	if expected := "/tmp/1979/Apocalypse.Now/"; item.LocalDir != expected {
		t.Fatalf("Expected %q, got %q", expected, item.LocalDir)
	}
}

func TestNewItemDefaultParser(t *testing.T) {
	s := Site{
		localDir: template.Must(template.New("").Parse("/tmp/")),
		parser:   parser.Default,
	}
	d := lftp.File{Path: "/foo/The.Wire.S03E01"}
	q := Queue{Site: s}
	item := newTestItem(&q, d)
	if expected := "/tmp/"; item.LocalDir != expected {
		t.Fatalf("Expected %s, got %s", expected, item.LocalDir)
	}
}

func TestNewItemUnparsable(t *testing.T) {
	tmpl := template.Must(template.New("").Parse(
		"/tmp/{{ .Name }}/S{{ .Season }}/"))
	s := Site{
		localDir: tmpl,
		parser:   parser.Show,
	}
	d := lftp.File{Path: "/foo/bar"}
	q := Queue{Site: s}
	item, err := newItem(&q, d)
	if err == nil {
		t.Fatal("Expected error")
	}
	if item.LocalDir != "" {
		t.Fatal("Expected empty string")
	}
	if item.Transfer {
		t.Fatal("Expected item to not be transferred")
	}
}

func TestNewItemWithReplacements(t *testing.T) {
	tmpl := template.Must(template.New("").Parse(
		"/tmp/{{ .Name }}/S{{ .Season }}/"))
	s := Site{
		localDir: tmpl,
		parser:   parser.Show,
		Replacements: []Replacement{
			Replacement{pattern: regexp.MustCompile("_"), Replacement: "."},
			Replacement{pattern: regexp.MustCompile("\\.Of\\."), Replacement: ".of."},
			Replacement{pattern: regexp.MustCompile("\\.the\\."), Replacement: ".The."},
			Replacement{pattern: regexp.MustCompile("\\.And\\."), Replacement: ".and."},
		},
	}
	q := Queue{Site: s}
	var tests = []struct {
		in  Item
		out string
	}{
		{newTestItem(&q, lftp.File{Path: "/foo/Game.Of.Thrones.S01E01"}), "Game.of.Thrones"},
		{newTestItem(&q, lftp.File{Path: "/foo/Fear.the.Walking.Dead.S01E01"}), "Fear.The.Walking.Dead"},
		{newTestItem(&q, lftp.File{Path: "/foo/Halt.And.Catch.Fire.S01E01"}), "Halt.and.Catch.Fire"},
		{newTestItem(&q, lftp.File{Path: "/foo/Top_Gear.01x01"}), "Top.Gear"},
	}
	for _, tt := range tests {
		if tt.in.Media.Name != tt.out {
			t.Errorf("Expected %q, got %q", tt.out, tt.in.Media.Name)
		}
	}

}

func TestWeight(t *testing.T) {
	s := Site{
		priorities: []*regexp.Regexp{regexp.MustCompile("\\.PROPER\\."), regexp.MustCompile("\\.REPACK\\.")},
	}
	q := Queue{Site: s}
	var tests = []struct {
		in  Item
		out int
	}{
		{Item{Queue: &q, Remote: lftp.File{Path: "/tmp/The.Wire.S01E01.foo"}}, 0},
		{Item{Queue: &q, Remote: lftp.File{Path: "/tmp/The.Wire.S01E01.PROPER.foo"}}, 2},
		{Item{Queue: &q, Remote: lftp.File{Path: "/tmp/The.Wire.S01E01.REPACK.foo"}}, 1},
	}
	for _, tt := range tests {
		if in := tt.in.Weight(); in != tt.out {
			t.Errorf("Expected %q, got %q", tt.out, in)
		}
	}
}

func TestItemsSort(t *testing.T) {
	items := Items{
		Item{Remote: lftp.File{Path: "/x/c"}},
		Item{Remote: lftp.File{Path: "/x/b"}},
		Item{Remote: lftp.File{Path: "/x/a"}},
		Item{Remote: lftp.File{Path: "/y/a"}},
	}
	sort.Sort(items)
	var tests = []struct {
		in  int
		out string
	}{
		{0, "/x/a"},
		{1, "/x/b"},
		{2, "/x/c"},
		{3, "/y/a"},
	}
	for _, tt := range tests {
		if got := items[tt.in].Remote.Path; got != tt.out {
			t.Errorf("Expected index %d to be %q, got %q", tt.in, tt.out, got)
		}
	}
}

func TestAccept(t *testing.T) {
	item := Item{}
	item.Accept("foo")
	if !item.Transfer {
		t.Error("Expected true")
	}
	if expected := "foo"; item.Reason != expected {
		t.Errorf("Expected %q, got %q", expected, item.Reason)
	}
}

func TestReject(t *testing.T) {
	item := Item{}
	item.Reject("bar")
	if item.Transfer {
		t.Error("Expected false")
	}
	if expected := "bar"; item.Reason != expected {
		t.Errorf("Expected %q, got %q", expected, item.Reason)
	}
}

func TestDstDir(t *testing.T) {
	var tests = []struct {
		in  Item
		out string
	}{
		{Item{Remote: lftp.File{Path: "/foo/bar"}, LocalDir: "/tmp/"}, "/tmp/bar"},
		{Item{Remote: lftp.File{Path: "/foo/bar"}, LocalDir: "/tmp/foo/bar"}, "/tmp/foo/bar"},
	}
	for _, tt := range tests {
		if got := tt.in.DstDir(); got != tt.out {
			t.Errorf("Expected %q, got %q", tt.out, got)
		}
	}
}

func TestIsEmpty(t *testing.T) {
	readDir := func(dirname string) ([]os.FileInfo, error) {
		if dirname == "/tmp/bar" {
			return []os.FileInfo{fileInfoStub{}}, nil
		}
		return nil, nil
	}
	var tests = []struct {
		in  Item
		out bool
	}{
		{Item{LocalDir: "/tmp/foo"}, true},
		{Item{LocalDir: "/tmp/bar"}, false},
	}
	for _, tt := range tests {
		if got := tt.in.IsEmpty(readDir); got != tt.out {
			t.Errorf("Expected %t, got %t", tt.out, got)
		}
	}
}

func TestDuplicates(t *testing.T) {
	tmpl := template.Must(template.New("").Parse(
		"/tmp/{{ .Name }}/S{{ .Season }}/"))
	s := Site{
		localDir:   tmpl,
		parser:     parser.Show,
		priorities: []*regexp.Regexp{regexp.MustCompile("\\.foo\\.")},
	}
	q := Queue{Site: s}
	readDir := func(dirname string) ([]os.FileInfo, error) {
		return []os.FileInfo{
			fileInfoStub{name: "The.Wire.S01E01.720p.BluRay.foo"},
			fileInfoStub{name: "The.Wire.S01E01.720p.BluRay.bar"},
			fileInfoStub{name: "The.Wire.S01E02.720p.BluRay.baz"},
		}, nil
	}
	item := newTestItem(&q, lftp.File{Path: "/tmp/The.Wire/S01/The.Wire.S01E01.720p.BluRay.foo"})
	items := item.duplicates(readDir)
	if l := len(items); l != 1 {
		t.Fatalf("Expected 1 duplicate, got %d", l)
	}
	if want := "The.Wire.S01E01.720p.BluRay.bar"; items[0].Media.Release != want {
		t.Errorf("Expected %q, got %+v", want, items[0].Media.Release)
	}
	if !items[0].Merged {
		t.Errorf("Expected Merged=true")
	}
	if !items[0].Transfer {
		t.Errorf("Expected Transfer=true")
	}
	if items[0].Media.IsEmpty() {
		t.Errorf("Expected non-empty media")
	}
	if items[0].Remote.Path != items[0].LocalDir {
		t.Errorf("Expected Path=%q and LocalDir=%q to be equal", items[0].Remote.Path, items[0].LocalDir)
	}
}