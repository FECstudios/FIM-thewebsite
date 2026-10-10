package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type resourceEntry struct {
	File string `json:"file"`
	Use  string `json:"use"`
}

func brandContext() string {
	var b strings.Builder

	if guide, err := os.ReadFile("resources/brand.md"); err == nil {
		b.WriteString("\n\nBRAND GUIDE (follow it exactly):\n")
		b.Write(guide)
	} else {
		log.Println("could not read brand.md :/")
	}

	data, err := os.ReadFile("resources/manifest.json")
	if err != nil {
		log.Println("Could not read manifest json: ", err)
		return b.String()
	}

	var entries []resourceEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		log.Println("manifest json is corrupted: ", err)
		return b.String()
	}

	var lines []string
	for _, e := range entries {
		if _, err := os.Stat(filepath.Join("resources", e.File)); err != nil {
			log.Println("exists but no reference in manifest: ", e.File)
			continue
		}

		lines = append(lines, fmt.Sprintf("- /resources/%s : %s", e.File, e.Use))
	}
	if len(lines) > 0 {
		b.WriteString("\n\nRESOURCES (the only files you may load, using these exact paths) :\n")
		b.WriteString(strings.Join(lines, "\n"))
	}

	return b.String()
}
