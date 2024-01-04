package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type Source struct {
	Source     string   `json:"source"`
	Paragraphs []string `json:"paragraphs"`
}

var outputTemplate = `package quotes

type Source struct {
	Source     string   
	Paragraphs []string 
}

var D = []Source{
{{ range . }}	{
		Source: "{{ .Source }}",
		Paragraphs: []string{
			{{ range .Paragraphs }}` + "`{{ . }}`" + `,
			{{ end }}},
	},
{{ end }}}
`

func main() {
	var jsons []string
	e := filepath.Walk(".",
		func(path string, info fs.FileInfo, e error) (e error) {
			if strings.HasSuffix(path, "json") {
				jsons = append(jsons, path)
			}
			return
		})
	if e != nil {
		_, _ = fmt.Fprintln(os.Stderr, "error:", e)
		os.Exit(1)
	}
	var quotes []Source
	var data []byte
	for i := range jsons {
		data, e = os.ReadFile(jsons[i])
		if e != nil {
			_, _ = fmt.Fprintln(os.Stderr, "error:", e)
			continue
		}
		var src Source
		e = json.Unmarshal(data, &src)
		if e != nil {
			_, _ = fmt.Fprintln(os.Stderr, "error:", e)
			continue
		}
		quotes = append(quotes, src)
	}
	outFile := "../quotes.go"
	tmpl, e := template.New(outFile).Parse(outputTemplate)
	if e != nil {
		_, _ = fmt.Fprintln(os.Stderr, "error:", e)
		os.Exit(1)
	}
	var fh *os.File
	fh, e = os.Create("../quotes_for_test.go")
	if e != nil {
		_, _ = fmt.Fprintln(os.Stderr, "error:", e)
		os.Exit(1)
	}
	e = tmpl.Execute(fh, quotes)
	if e != nil {
		_, _ = fmt.Fprintln(fh, "error:", e)
		os.Exit(1)
	}
}
