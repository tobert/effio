package effio

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"
)

// effio processes fio configs as go templates rather than using fio's built-in
// variable expansion to allow generated files to be 100% static
// the parent environment from effio will be intact when fio runs so there's no
// reason base fio configs can't use envvars, but it does make reproducing
// results less accurate
type FioConfTmpl struct {
	Filename string // full path to the source file
	Name     string // used to generate output filenames
	tmpl     *template.Template
}

type FioConfTmpls []FioConfTmpl

// Load loads all files in fts.SrcDir ending with ".fio" as Go templates.
// Template directives are optional. A plain fio config will pass through
// unharmed.
// Templates are parsed but not executed. effio.Suite calls Execute() directly.
func LoadFioConfDir(dir string) (fts FioConfTmpls) {
	visitor := func(fpath string, f os.FileInfo, err error) error {
		if err != nil {
			log.Fatalf("Encountered an error while loading fio config '%s': %s", fpath, err)
		}

		fname := path.Base(fpath)
		ext := path.Ext(fname)
		if ext == ".fio" {
			data, err := ioutil.ReadFile(fpath)
			if err != nil {
				log.Fatalf("Could not read fio config '%s': %s", fpath, err)
			}

			// remove the .fio to get the base filename to use as a generic name string
			name := strings.TrimSuffix(fname, ext)
			tmpl := template.Must(template.New(name).Parse(string(data)))

			fts = append(fts, FioConfTmpl{fpath, name, tmpl})
		}

		return nil
	}

	err := filepath.Walk(dir, visitor)
	if err != nil {
		log.Fatalf("Could not load configs in '%s': %s", dir, err)
	}

	return fts
}
