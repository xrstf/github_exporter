// SPDX-FileCopyrightText: 2023 Christoph Mewes
// SPDX-License-Identifier: MIT

package main

import (
	"bytes"
	"go/format"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

const (
	fields   = 100
	filename = "pkg/client/client_gen.go"
)

func makeRange(min, max int) []int {
	a := make([]int, max-min+1)
	for i := range a {
		a[i] = min + i
	}
	return a
}

func main() {
	templates, err := filepath.Glob("hack/*.go.tmpl")
	if err != nil {
		log.Fatalf("Failed to find Go templates: %v", err)
	}

	data := map[string]interface{}{
		"numFields": fields,
		"fields":    makeRange(0, fields-1),
	}

	for _, templateFile := range templates {
		log.Printf("Rendering %s...", templateFile)

		content, err := os.ReadFile(templateFile)
		if err != nil {
			log.Fatalf("Failed to read client_gen.go.tmpl -- did you run this from the root directory?: %v", err)
		}

		tpl := template.Must(template.New("tpl").Parse(string(content)))

		var buf bytes.Buffer
		tpl.Execute(&buf, data)

		source, err := format.Source(buf.Bytes())
		if err != nil {
			log.Fatalf("Failed to format generated code: %v", err)
		}

		filename := filepath.Join("pkg/client", strings.TrimSuffix(filepath.Base(templateFile), ".tmpl"))

		err = os.WriteFile(filename, source, 0644)
		if err != nil {
			log.Fatalf("Failed to write %s: %v", filename, err)
		}
	}
}
