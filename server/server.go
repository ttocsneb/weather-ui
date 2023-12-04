package server

import (
	"embed"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"text/template"

	"github.com/gorilla/mux"
	"github.com/ttocsneb/weather-ui/util"
)

//go:embed templates/*
var templFiles embed.FS
var templs map[string]*template.Template

func makeDict(values ...any) (map[string]any, error) {
	if len(values)%2 != 0 {
		return nil, errors.New("Invalid dict call")
	}
	dict := make(map[string]any)
	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			return nil, errors.New("Dict keys must be strings")
		}
		dict[key] = values[i+1]
	}
	return dict, nil
}

func loadTemplates() error {
	layouts, err := ReadDirRecursive(templFiles, "templates/layouts")
	includes, err := ReadDirRecursive(templFiles, "templates/includes")
	if err != nil {
		return err
	}
	templs = make(map[string]*template.Template)

	funcMap := template.FuncMap{
		"dict":   makeDict,
		"encode": util.EncodeURIString,
	}

	for _, layout := range layouts {
		name := strings.Replace(layout, "templates/layouts/", "", 1)
		files := append(includes, layout)
		templ, err := template.New("").Funcs(funcMap).ParseFS(templFiles, files...)
		if err != nil {
			return err
		}
		templs[name] = templ
	}

	include_templ, err := template.New("").Funcs(funcMap).ParseFS(templFiles, includes...)
	if err != nil {
		return err
	}
	for _, include := range includes {
		name := strings.Replace(include, "templates/includes/", "", 1)
		templs[name] = include_templ
	}
	return nil
}

type doneWriter struct {
	http.ResponseWriter
	done bool
}

func (w *doneWriter) WriteHeader(status int) {
	w.done = true
	w.ResponseWriter.WriteHeader(status)
}

func (w *doneWriter) Write(b []byte) (int, error) {
	w.done = true
	return w.ResponseWriter.Write(b)
}

func (w *doneWriter) Flush() {
	w.ResponseWriter.(http.Flusher).Flush()
}

/*
Create a Handler from a function that may fail. If the function fails, then a
500 error will be sent and the error logged. If the response has already
started then only the error log will happen.
*/
func HandlerFuncError(fn func(http.ResponseWriter, *http.Request) error) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dw := &doneWriter{ResponseWriter: w}
		err := fn(dw, r)
		if err == nil {
			return
		}
		if !dw.done {
			for k := range dw.Header() {
				delete(dw.Header(), k)
			}

			e := err.Error()
			if len(e) >= 4 {
				code, conv_err := strconv.Atoi(e[:3])
				if conv_err == nil {
					dw.Header().Set("Content-Type", "text/html")
					dw.WriteHeader(code)
					dw.Write([]byte(fmt.Sprintf("<p>%v: %v</p>", code, e[4:])))

					goto end
				}
			}

			dw.Header().Set("Content-Type", "text/html")
			dw.WriteHeader(500)
			dw.Write([]byte("<p>500: Internal Server Error</p>"))
		}
	end:
		fmt.Printf("Error on `%v`: %v\n", r.URL.Path, err)
	})
}

func RenderTemplate(response io.Writer, name string, vars any) error {
	templ, exists := templs[name]
	if !exists {
		return fmt.Errorf("Template %v does not exist", name)
	}
	buf := util.BufPool.Get()
	defer util.BufPool.Put(buf)

	err := templ.ExecuteTemplate(buf, name, vars)
	if err != nil {
		return err
	}

	data := util.EscapeHtmlNewlines(buf.String())

	response.Write([]byte(data))

	return nil
}

func Serve(conf util.Config) error {
	err := loadTemplates()
	if err != nil {
		return err
	}

	r := mux.NewRouter()

	RootRoutes(r, &conf)
	StationRoutes(r, &conf)
	RegionRoutes(r, &conf)
	LocationRoutes(r, &conf)

	fmt.Printf("Starting server on port %v\n", conf.Port)

	return http.ListenAndServe(fmt.Sprintf(":%v", conf.Port), r)
}
