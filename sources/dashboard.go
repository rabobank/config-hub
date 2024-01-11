package sources

import (
	"net/http"
	"text/template"

	"github.com/gomatbase/go-we"
)

type SourceDashboardReport struct {
	Name    string
	RawHtml *string
}

var dashboardTemplate = template.Must(template.New("dashboard").Parse("<!DOCTYPE html>\n" +
	"<html lang=\"en\">\n" +
	"    <head>\n" +
	"        <meta charset=\"UTF-8\">\n" +
	"        <title>Config Hub Dashboard</title>\n" +
	"        <style>\n" +
	"            body {\n" +
	"                background-color: #f5f5f5;\n" +
	"                font-family: \"Helvetica Neue\", Helvetica, Arial, sans-serif;\n" +
	"                font-size: 14px;\n" +
	"                line-height: 1.42857143;\n" +
	"                color: #333;\n" +
	"            }\n" +
	"        </style>\n" +
	"    </head>\n" +
	"    <body>\n" +
	"        <div class=\"report\">\n" +
	"            <h1 class=\"title\">Config Hub Dashboard</h1>\n" +
	"{{range .}}" +
	"            <div class=\"source\">\n" +
	"                <h2 class=\"source-title\"><span class=\"label\">Source</span>&nbsp;<span class=\"value\">{{.Name}}</span></h2>\n" +
	"{{if .RawHtml}}" +
	"                <div class=\"source-report\">\n" +
	"{{.RawHtml}}" +
	"                </div>\n" +
	"{{end}}" +
	"{{end}}"))

func Dashboard(writer we.ResponseWriter, _ we.RequestScope) error {
	writer.WriteHeader(http.StatusOK)

	sourceDashboards := make([]SourceDashboardReport, len(propertySources))
	for i, s := range propertySources {
		sourceDashboards[i] = SourceDashboardReport{Name: s.Name(), RawHtml: s.DashboardReport()}
	}

	return dashboardTemplate.Execute(writer, sourceDashboards)
}
