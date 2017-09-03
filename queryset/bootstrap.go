package queryset

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/jirfag/go-queryset/parser"
	"golang.org/x/tools/go/loader"
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
	var outF *os.File
	outF, err := os.OpenFile(outFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0640)
	if err != nil {
		return fmt.Errorf("can't open out file: %s", err)
	}
	defer outF.Close()

	const hdrTmpl = `package %s
  import (
    "github.com/jinzhu/gorm"
    "github.com/jirfag/go-queryset/queryset/base"
  )
`
	_, err = outF.WriteString(fmt.Sprintf(hdrTmpl, pkgInfo.Pkg.Name()))
	if err != nil {
		return fmt.Errorf("can't write hdr string into out file: %s", err)
	}
	if _, err = io.Copy(outF, r); err != nil {
		return fmt.Errorf("can't write to out file: %s", err)
	}

	cmd := exec.Command("goimports", "-w", outF.Name())
	if err = cmd.Run(); err != nil {
		return fmt.Errorf("can't execute goimports on outfile: %s", err)
	}

	return nil
}
