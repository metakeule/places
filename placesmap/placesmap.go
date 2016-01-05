/*
package placesmap provides a places.Mapper to allow embedding of templates, escaping etc
*/
package placesmap

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/metakeule/places"
	"html"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

// split splits the given input and returns the prefix and rest.
// If the first character is a minus sign ("-") then everything that follows
// up to the next whitespace ("\s") is considered to be the prefix and everything
// following that whitespace is considered to be the rest
// If the first character is no minus sign, the prefix is empty and the rest equals the
// input.
// If input is the empty string or just the minus sign, prefix and rest are empty strings.
func split(input string) (prefix string, rest string) {
	if input == "" || input == "-" {
		return
	}

	if input[0] != '-' {
		rest = input
		return
	}

	idx := strings.IndexRune(input, ' ')

	if idx == -1 {
		prefix = input[1:]
		return
	}

	prefix = input[1:idx]
	rest = strings.TrimSpace(input[idx:])
	return
}

// Map is a registry of places.Mappers and itself a places.Mapper
type Map interface {
	places.Mapper

	// Add registers a mapper in the registry for the given prefix
	// If there is already a mapper for the given prefix, it will be overwritten
	// If prefix does not conform to the regular expression ^[a-z]+$, ErrInvalidPrefix is returned
	Add(prefix string, mapper places.Mapper) error
}

var ErrInvalidPrefix = errors.New("prefix does not match the regular expression ^[a-z]+$")

type MapperAlreadyExistsError string

func (m MapperAlreadyExistsError) Error() string {
	return fmt.Sprintf("Mapper for prefix %#v already exists", m)
}

var prefixRule = regexp.MustCompile("^[a-z]+$")

type _map map[string]places.Mapper

// Add registers a mapper in the registry for the given prefix
// If there is already a mapper for the given prefix, it will be overwritten
// If prefix is the empty string, the default mapper is set.
// If prefix does not conform to the regular expression ^[a-z]+$, ErrInvalidPrefix is returned
// If a mapper already exists for this prefix, MapperAlreadyExistsError is returned
func (mp _map) Add(prefix string, mapper places.Mapper) error {
	if prefix != "" && !prefixRule.MatchString(prefix) {
		return ErrInvalidPrefix
	}
	if _, has := mp[prefix]; has {
		return MapperAlreadyExistsError(prefix)
	}
	mp[prefix] = mapper
	return nil
}

// Map returns the value for the given input
// If input has a prefix, i.e. starts with minus sign and
// conforms to the regular expression ^[a-z]+$, the corresponding
// mapper, registered via Add is used. Without a prefix, the default mapper
// (registered for prefix == "") is used.
// If a prefix exists but not mapper can be found, the empty string is returned
func (mp _map) Map(input string) string {
	prefix, rest := split(input)

	// fmt.Printf("prefix: %#v rest: %#v", prefix, rest)

	if prefix == "" && rest == "" {
		return ""
	}

	// If a prefix is identified but does not conform to prefixRule,
	// an empty string will be returned, since Add makes sure the every prefix
	// in the map conforms
	m, ok := mp[prefix]

	if !ok {
		return ""
	}

	return m.Map(rest)
}

// Empty is a places.Mapper that always returns an empty string
type Empty struct{}

func (e Empty) Map(string) string {
	return ""
}

// Empty is a places.Mapper that always returns the string itself
type String string

func (s String) Map(string) string {
	return string(s)
}

type Self string

// Self is a places.Mapper that always returns the placeholder prefixed by the value of Self
func (s Self) Map(placeholder string) string {
	return string(s) + placeholder
}

// MapFunc is a func implementing places.Mapper
type MapFunc func(string) string

func (m MapFunc) Map(placeholder string) string {
	return m(placeholder)
}

// ReadSeekerMap is a map of strings to io.ReadSeeker that may be used concurrently
type ReadSeekerMap struct {
	mx sync.RWMutex
	m  map[string]io.ReadSeeker
}

// NewReadSeekerMap returns a
func NewReadSeekerMap() *ReadSeekerMap {
	return &ReadSeekerMap{
		m: map[string]io.ReadSeeker{},
	}
}

type ReadSeekerAlreadyExistsError string

func (m ReadSeekerAlreadyExistsError) Error() string {
	return fmt.Sprintf("ReadSeeker for name %#v already exists", m)
}

// Add adds an io.ReadSeeker for the given name.
// If there is already a ReadSeeker defined for the given name,
// an error is returned.
// There are no restrictions for the name
func (r *ReadSeekerMap) Add(name string, rs io.ReadSeeker) error {
	if _, has := r.m[name]; has {
		return ReadSeekerAlreadyExistsError(name)
	}
	r.m[name] = rs
	return nil
}

func (r *ReadSeekerMap) Map(name string) (val string) {
	r.mx.RLock()
	if rs, ok := r.m[name]; ok {
		_, err := rs.Seek(0, 0)
		if err == nil {
			var b []byte
			b, err = ioutil.ReadAll(rs)
			if err == nil {
				val = string(b)
			}
		}
	}

	r.mx.RUnlock()
	return
}

// TemplateLoader loads templates recursively from a root directory for a given file extension
type TemplateLoader struct {
	*ReadSeekerMap
	rootDir    string
	extension  string
	ignoreDirs *regexp.Regexp // directories to be ignored
}

type RootIsNotDirectoryError string

func (r RootIsNotDirectoryError) Error() string {
	return fmt.Sprintf("root %#v is not a directory", r)
}

type RootDoesNotExistError string

func (r RootDoesNotExistError) Error() string {
	return fmt.Sprintf("root %#v does not exist", r)
}

func (l *TemplateLoader) fileReader(path string) (io.ReadSeeker, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(b), nil
}

func (l *TemplateLoader) walk(path string, info os.FileInfo, err error) error {
	if err != nil {
		return err
	}

	if l.ignoreDirs != nil && info.IsDir() && l.ignoreDirs.MatchString(info.Name()) {
		return filepath.SkipDir
	}

	if !info.IsDir() && filepath.Ext(path) == l.extension {
		var rel string
		rel, err = filepath.Rel(l.rootDir, path)
		if err != nil {
			return err
		}
		var rd io.ReadSeeker
		rd, err = l.fileReader(path)
		if err != nil {
			return err
		}
		l.ReadSeekerMap.Add(rel, rd)
	}
	return nil
}

func (l *TemplateLoader) Load() (*ReadSeekerMap, error) {
	info, errStat := os.Stat(l.rootDir)

	if errStat != nil {
		if os.IsNotExist(errStat) {
			return nil, RootDoesNotExistError(l.rootDir)
		}
		return nil, errStat
	}

	if !info.IsDir() {
		return nil, RootIsNotDirectoryError(l.rootDir)
	}

	l.ReadSeekerMap = NewReadSeekerMap()

	err := filepath.Walk(l.rootDir, l.walk)

	if err != nil {
		return nil, err
	}

	return l.ReadSeekerMap, nil
}

type HTMLTemplate struct {
	sync.RWMutex
	rs  *ReadSeekerMap
	rsm map[string]*places.Template
}

func NewHTMLTemplate(rs *ReadSeekerMap) *HTMLTemplate {
	h := &HTMLTemplate{
		rs:  rs,
		rsm: map[string]*places.Template{},
	}
	h.rs.mx.RLock()

	for k, rs := range h.rs.m {
		_, err := rs.Seek(0, 0)
		if err == nil {
			var b []byte
			b, err = ioutil.ReadAll(rs)
			if err == nil {
				h.rsm[k] = places.NewTemplate(b)
			}
		}
	}
	h.rs.mx.RUnlock()
	return h
}

func (h *HTMLTemplate) NewMapper(m map[string]places.Mapper) *HTMLTemplateMapper {
	return &HTMLTemplateMapper{HTMLTemplate: h, m: m}
}

type HTMLTemplateMapper struct {
	sync.Mutex
	*HTMLTemplate
	m         map[string]places.Mapper
	preferred places.Mapper
	indexes   []NMapper // keep track of array indexes within nested objects
	depth     int       // current depth of nested objects
}

func (h *HTMLTemplateMapper) require(name string, m places.Mapper) string {
	// fmt.Printf("requiring: %#v\n", name)
	h.HTMLTemplate.RLock()
	defer h.HTMLTemplate.RUnlock()

	if t, ok := h.HTMLTemplate.rsm[name]; ok {
		var bf bytes.Buffer
		t.ReplaceMapper(&bf, m)
		return bf.String()
	}
	return ""
}

type NMapper interface {
	places.Mapper
	NMap(n int, sub string) places.Mapper
	Len() int
}

type mapDelegate struct {
	first, second places.Mapper
}

func (m *mapDelegate) Map(input string) string {
	out := m.first.Map(input)
	if out != "" {
		return out
	}
	return m.second.Map(input)
}

/*
func (h *HTMLTemplateMapper) _map_delegate(input string, m places.Mapper) string {
	out := m.Map(input)
	if out != "" {
		return out
	}
	return h._map(input)
}
*/

func (h *HTMLTemplateMapper) Map(input string) string {
	if h.preferred != nil {
		out := h.preferred.Map(input)
		if out != "" {
			return out
		}
	}
	return h._map(input)
}

func (h *HTMLTemplateMapper) findMapper(depth int) NMapper {
	h.Lock()
	defer h.Unlock()
	if len(h.indexes) > depth {
		return nil
	}
	return h.indexes[depth-1]
}

// findNestedMapper finds a mapper for a nested object
func (h *HTMLTemplateMapper) findNestedMapper(sub string) places.Mapper {
	fmt.Printf("inside findNestedMapper: %#v, depth: %d\n", sub, h.depth)
	if sub == "" {
		return String("")
	}
	sb := strings.Split(sub, ".")

	if len(sb) != h.depth {
		return String("[Error] too deep var declaration")
	}

	var m places.Mapper

	for d := 0; d <= len(sb); d++ {
		nm := h.findMapper(d)

		if nm == nil {
			return m
		}

		m = nm
	}

	return m
}

func (h *HTMLTemplateMapper) replaceVars(bf places.Buffer, t *places.Template, nm NMapper, sub string) {
	fmt.Printf("replaceVars for mapper %#v, sub: %#v\n", nm, sub)
	/*
		if sub != "" {
			for i := 0; i < l; i++ {
				var m = nm.NMap(i, sub)
				if nmm, isNM := m.(NMapper); isNM {
					h.replaceVars(bf, t, nmm, sub)
					continue
				}
				h.preferred = nm
				t.ReplaceMapper(bf, h)
				h.preferred = nil
			}
			return
		}
	*/
	/*
		idx := strings.IndexRune(sub, '.')
		if idx == -1 {
			sub = ""
		} else {
			sub = sub[idx+1:]
		}
	*/

	// h.depth++

	l := nm.Len()
	for i := 0; i < l; i++ {
		var m = nm.NMap(i, sub)
		if nmm, isNM := m.(NMapper); isNM {
			h.replaceVars(bf, t, nmm, sub)
		} else {
			fmt.Printf("got mapper: %#v[%d]\n", m, i)
			h.preferred = m
			t.ReplaceMapper(bf, h)
			h.preferred = nil
		}
	}
	/*
		// h.depth--
		fmt.Printf("indexes: %#v, depth: %d, sub: %#v\n", h.indexes, h.depth, sub)
		if h.depth == 1 && sub != "" {
			fmt.Printf("now calling findNestedMapper\n")
			h.depth = len(strings.Split(sub, "."))
			h.preferred = h.findNestedMapper(sub)
			fmt.Printf("found nested mapper: %#v\n", h.preferred)
			t.ReplaceMapper(bf, h)
			h.indexes = []NMapper{}
			h.depth = 0
		}
		h.preferred = nil
	*/
}

func (h *HTMLTemplateMapper) _map(input string) string {
	prefix, rest := split(input)

	fmt.Printf("prefix: %#v rest: %#v\n", prefix, rest)
	if prefix == "require" {
		return h.require(rest, h)
	}

	if prefix == "each" {
		s := strings.SplitN(rest, " ", 2)
		mpName, inc := strings.TrimSpace(s[0]), strings.TrimSpace(s[1])
		var sub string

		if strings.ContainsRune(mpName, '.') {
			sp := strings.SplitN(mpName, ".", 2)
			mpName = sp[0]
			sub = sp[1]
		}

		fmt.Printf("mpName: %#v, inc: %#v\n", mpName, inc)
		h.Lock()
		mp, ok := h.m[mpName]
		h.Unlock()
		if !ok {
			fmt.Printf("mpName %#v not found", mpName)
			return ""
		}

		h.HTMLTemplate.RLock()
		t, hasTemplate := h.HTMLTemplate.rsm[inc]
		h.HTMLTemplate.RLock()
		if !hasTemplate {
			fmt.Printf("template %#v not found", inc)
			return ""
		}

		// TODO: debug this properly
		if nm, is := mp.(NMapper); is {
			h.depth++
			h.indexes = append(h.indexes, NMapper(nil))
			var bf bytes.Buffer
			l := nm.Len()
			for i := 0; i < l; i++ {
				var m = nm.NMap(i, sub)
				fmt.Printf("got mapper: %#v[%d]\n", m, i)
				if nmm, isNM := m.(NMapper); isNM {
					h.indexes[h.depth-1] = nmm
					h.replaceVars(&bf, t, nmm, sub)
				} else {
					// h.depth--
					// fmt.Printf("indexes: %#v, depth: %d, sub: %#v\n", h.indexes, h.depth, sub)
					if sub != "" {
						fmt.Printf("now calling findNestedMapper\n")
						h.depth = len(strings.Split(sub, "."))
						h.preferred = h.findNestedMapper(sub)
						fmt.Printf("found nested mapper: %#v\n", h.preferred)
						t.ReplaceMapper(&bf, h)
						h.indexes = []NMapper{}
						h.depth = 0
					} else {

						h.preferred = nil
						h.preferred = m
						t.ReplaceMapper(&bf, h)
						h.preferred = nil
					}
				}
			}

			h.replaceVars(&bf, t, nm, sub)
			h.preferred = nil
			h.indexes = h.indexes[:len(h.indexes)-1]
			h.depth--
			return bf.String()
		} else {
			return fmt.Sprintf("not a NMapper: %#v\n", mp)
		}
	}

	h.Lock()
	mp, ok := h.m[rest]
	h.Unlock()
	if !ok {
		return ""
	}

	switch prefix {
	case "js":
		return fmt.Sprintf("%#v", mp.Map(rest))
	case "raw":
		return mp.Map(rest)
	case "html":
		return mp.Map(rest)
	case "url":
		return url.QueryEscape(mp.Map(rest))
	case "include":
		if val := mp.Map(strings.TrimSpace(rest)); val != "" {
			return h.require(val, h)
		}
		return ""
	default:
		return html.EscapeString(mp.Map(rest))
	}

}

/*
func NewHTMLTemplates(rootDir string, ignoreDirs *regexp.Regexp, m map[string]string) (places.Mapper, error) {
	l := NewTemplateLoader(rootDir, ".html", ignoreDirs)
	rs, err := l.Load()
	if err != nil {
		return nil, err
	}

	t := NewHTMLTemplate(rs, m)

	return t, nil
}
*/

func NewTemplateLoader(rootDir string, extension string, ignoreDirs *regexp.Regexp) *TemplateLoader {
	return &TemplateLoader{
		rootDir:    rootDir,
		extension:  extension,
		ignoreDirs: ignoreDirs,
	}
}

var HTMLEscape = MapFunc(html.EscapeString)
var UrlEscape = MapFunc(url.QueryEscape)

// New returns a new Map that is not safe for concurrent use.
func New() Map {
	return _map{}
}

// NewConcurrent returns a new Map that is safe for concurrent use.
func NewConcurrent() Map {
	return &_map_concurrent{m: _map{}}
}

type _map_concurrent struct {
	mx sync.RWMutex
	m  _map
}

func (c *_map_concurrent) Add(prefix string, mapper places.Mapper) error {
	c.mx.Lock()
	err := c.m.Add(prefix, mapper)
	c.mx.Unlock()
	return err
}

func (c *_map_concurrent) Map(input string) string {
	c.mx.RLock()
	res := c.m.Map(input)
	c.mx.RUnlock()
	return res
}
