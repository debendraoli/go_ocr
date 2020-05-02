package helpers

import (
	"fmt"
	"github.com/otiai10/gosseract"
	"go_ocr/helpers/ghostscript"
	"io/ioutil"
	"os"
	"strings"
)

func ConvertToPNG(pdfPath *os.File, filesChan chan map[string][]string) {

	tmpIndex := strings.LastIndex(pdfPath.Name(), "/")
	tmpDir := pdfPath.Name()[:tmpIndex]
	pdfFile := pdfPath.Name()
	args := []string{
		"-dNOPAUSE",
		"-dBATCH",
		"-dTextAlphaBits=4",
		"-dGraphicsAlphaBits=4",
		"-r300",
		"-sDEVICE=pngmonod",
		"-o",
		fmt.Sprintf("%s/%%03d.png", tmpDir),
		pdfFile,
	}

	gs, err := ghostScript()
	if err != nil {
		return
	}
	defer func() {
		_ = os.Remove(pdfFile)
		pdfPath.Close()
		_ = gs.Exit()
		gs.Destroy()
		filesConverted, _ := ioutil.ReadDir(tmpDir)
		files := make(map[string][]string)
		for _, file := range filesConverted {
			files[tmpDir] = append(files[tmpDir], tmpDir + "/" + file.Name())
		}
		filesChan <- files
	}()
	if err := gs.Init(args); err != nil {
		return
	}
	return
}

func ImageProcessor(imagePath *os.File, filesChan chan map[string][]string) {
	tmpIndex := strings.LastIndex(imagePath.Name(), "/")
	tmpDir := imagePath.Name()[:tmpIndex]
	ImageFile := imagePath.Name()
	files := map[string][]string{
		tmpDir: {ImageFile},
	}
	filesChan <- files
	return
}

func GetEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}

func ghostScript() (*ghostscript.Ghostscript, error)  {
	gs, err := ghostscript.NewInstance()
	if err != nil {
		return nil, err
	}
	return gs, nil
}


func Client() *gosseract.Client {
	client := gosseract.NewClient()
	return client
}

