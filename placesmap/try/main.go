package main

import (
	"bytes"
	"fmt"
	"github.com/go-on/stack"
	"github.com/metakeule/places"
	"github.com/metakeule/places/placesmap"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
)

var ignoreDirs = regexp.MustCompile("^.git$")

type templates struct {
	tt           *placesmap.HTMLTemplate
	mainTemplate *places.Template
}

func (t *templates) Reclaim(other interface{}) {
	*t = *(other.(*templates)) // will never panic
}

func (t *templates) Replace(bf places.Buffer, m map[string]places.Mapper) {
	t.mainTemplate.ReplaceMapper(bf, t.tt.NewMapper(m))
}

func New(mainTemplatePath string, includesDir string, extension string, ignoreDirs *regexp.Regexp) (*templates, error) {
	l := placesmap.NewTemplateLoader(includesDir, extension, ignoreDirs)

	rs, err := l.Load()
	if err != nil {
		return nil, err
	}

	t := &templates{}
	t.tt = placesmap.NewHTMLTemplate(rs)
	template, err2 := ioutil.ReadFile(mainTemplatePath)
	if err2 != nil {
		return nil, err2
	}

	t.mainTemplate = places.NewTemplate(template)
	return t, err
}

//var t *templates

type templateServer struct {
	*templates
}

func (t *templateServer) ServeHTTP(ctx stack.Contexter, rw http.ResponseWriter, rq *http.Request, next http.Handler) {
	ctx.Set(t.templates)
	m := mapCtx{}
	ctx.Set(&m)
	next.ServeHTTP(rw, rq)
}

type mapCtx map[string]places.Mapper

func (m *mapCtx) Reclaim(o interface{}) {
	*m = *(o.(*mapCtx))
}

func serve1(ctx stack.Contexter, rw http.ResponseWriter, rq *http.Request, next http.Handler) {
	var m mapCtx
	ctx.Get(&m)
	m["header.title"] = placesmap.String("soso")
	next.ServeHTTP(rw, rq)
}

type Company struct {
	Name string
}

func (c *Company) Map(in string) string {
	fmt.Println("COMPANY Map called")
	switch in {
	case "name":
		return c.Name
	default:
		return ""
	}
}

type Companies []*Company

func (c Companies) Map(string) string {
	return ""
}

func (c Companies) NMap(n int, sub string) places.Mapper {
	fmt.Println("COMPANIES NMap called")
	if n > len(c)-1 {
		return placesmap.String("")
	}
	return c[n]
}

func (c Companies) Len() int {
	fmt.Println("COMPANIES Len called")
	return len(c)
}

type User struct {
	Firstname, Lastname, ID string
	Companies               Companies
}

func (u *User) Map(in string) string {
	fmt.Println("USER Map with " + in)
	switch in {
	case "firstname":
		return u.Firstname
	case "lastname":
		return u.Lastname
	case "id":
		return u.ID
	default:
		return ""
	}
}

type Users []*User

func (u Users) Map(string) string {
	return ""
}

func (u Users) NMap(n int, sub string) places.Mapper {
	if n > len(u)-1 {
		return placesmap.String("")
	}
	fmt.Printf("USERS NMap called for [%d]%#v\n", n, sub)
	if sub == "companies" {
		return u[n].Companies
	}

	return u[n]
}

func (u Users) Len() int {
	return len(u)
}

func serve2(ctx stack.Contexter, rw http.ResponseWriter, rq *http.Request, next http.Handler) {
	var m mapCtx
	ctx.Get(&m)
	m["myurl"] = placesmap.String(rq.URL.Path)
	m["myclass"] = placesmap.String("<hiho>")

	m["mybody"] = placesmap.String("<bleib>")

	var c1 = Companies{
		{"Walt Disney"},
	}

	var c2 = Companies{
		{"Universal"},
	}

	var u = Users{
		{"Donald", "Duck", "1", c1},
		{"Mickey", "Mouse", "2", c2},
	}

	m["users"] = u
	m["show-user-delete"] = placesmap.String("user-delete.html")
	m["user-add"] = placesmap.String("user-add.html")
	ctx.Set(&m)
	next.ServeHTTP(rw, rq)
}

func finish(ctx stack.Contexter, rw http.ResponseWriter, rq *http.Request) {
	var t templates
	ctx.Get(&t)
	var m mapCtx
	ctx.Get(&m)
	var bf bytes.Buffer
	t.Replace(&bf, m)
	bf.WriteTo(rw)
}

func main() {

	wd, _ := os.Getwd()
	t, err := New(filepath.Join(wd, "index.html"), filepath.Join(wd, "includes"), ".html", ignoreDirs)
	if err != nil {
		println(err.Error())
		os.Exit(1)
	}

	s := stack.New()
	s.UseWithContext(&templateServer{t})
	s.UseFuncWithContext(serve1)
	s.UseFuncWithContext(serve2)
	s.UseHandlerFuncWithContext(finish)

	println("listening on port 3000")
	http.ListenAndServe(":3000", s.HandlerWithContext())

}
