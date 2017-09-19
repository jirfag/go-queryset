package queryset

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/jirfag/go-queryset/parser"
	"golang.org/x/tools/go/loader"
	"golang.org/x/tools/imports"
)

// GenerateQuerySets generates output file with querysets
func GenerateQuerySets(inFilePath, outFilePath string) error {
	pkgInfo, structs, err := parser.GetStructsInFile(inFilePath)
	if err != nil {
		return fmt.Errorf("can't parse file %s to get structs: %s", inFilePath, err)
	}

	var r io.Reader
	r, err = GenerateQuerySetsForStructs(pkgInfo, structs)
	if err != nil {
		return fmt.Errorf("can't generate query sets: %s", err)
	}

	if r == nil {
		return fmt.Errorf("no structs to generate query set in %s", inFilePath)
	}

	if err = writeQuerySetsToOutput(r, pkgInfo, outFilePath); err != nil {
		return fmt.Errorf("can't save query sets to out file %s: %s", outFilePath, err)
	}

	var absOutPath string
	absOutPath, err = filepath.Abs(outFilePath)
	if err != nil {
		absOutPath = outFilePath
	}

	log.Printf("successfully wrote querysets to %s", absOutPath)
	return nil
}

func writeQuerySetsToOutput(r io.Reader, pkgInfo *loader.PackageInfo, outFile string) error {
	const hdrTmpl = `package %s

import (
	"fmt"
	"time"

	"github.com/jinzhu/gorm"
)
`

	var buf bytes.Buffer
	pkgName := fmt.Sprintf(hdrTmpl, pkgInfo.Pkg.Name())
	if _, err := buf.WriteString(pkgName); err != nil {
		return fmt.Errorf("can't write hdr string into buf: %s", err)
	}
	if _, err := io.Copy(&buf, r); err != nil {
		return fmt.Errorf("can't write to buf: %s", err)
	}

	formattedRes, err := imports.Process(outFile, buf.Bytes(), nil)
	if err != nil {
		return fmt.Errorf("can't format generated file: %s", err)
	}

	var outF *os.File
	outF, err = os.OpenFile(outFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0640)
	if err != nil {
		return fmt.Errorf("can't open out file: %s", err)
	}
	defer func() {
		if e := outF.Close(); e != nil {
			log.Printf("can't close file: %s", e)
		}
	}()

	if _, err = outF.Write(formattedRes); err != nil {
		return fmt.Errorf("can't write to out file: %s", err)
	}

	return nil
}
