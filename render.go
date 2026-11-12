package main

import (
	_ "embed"
	"encoding/json"
	"html/template"
	"io"
)

//go:embed report.html.tmpl
var reportHTML string

var reportTemplate = template.Must(template.New("report").Parse(reportHTML))

func renderHTML(w io.Writer, r report) error {
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}
	return reportTemplate.Execute(w, struct {
		ReportJSON template.JS
	}{ReportJSON: template.JS(data)})
}
