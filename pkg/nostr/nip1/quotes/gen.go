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

var outputTemplate string = `package nip1_test

type Source struct {
	Source     string   
	Paragraphs []string 
}

var quotes = []Source{
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
	err := filepath.Walk(".",
		func(path string, info fs.FileInfo, err error) (e error) {
			if strings.HasSuffix(path, "json") {
				jsons = append(jsons, path)
			}
			return
		})
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	var quotes []Source
	var data []byte
	for i := range jsons {
		data, err = os.ReadFile(jsons[i])
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, "error:", err)
			continue
		}
		var src Source
		err = json.Unmarshal(data, &src)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, "error:", err)
			continue
		}
		quotes = append(quotes, src)
	}
	outFile := "../quotes.go"
	tmpl, err := template.New(outFile).Parse(outputTemplate)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	var fh *os.File
	fh, err = os.Create("../quotes_for_test.go")
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	err = tmpl.Execute(fh, quotes)
	if err != nil {
		_, _ = fmt.Fprintln(fh, "error:", err)
		os.Exit(1)
	}
}
