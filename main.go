package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

const chunkSize = 64000
const exitStatusCopyUnsafe = 3
const exitStatusOtherErrors = 4

func readDirIntoMap(path string) map[string]os.DirEntry {
	res := make(map[string]os.DirEntry)
	entries, err := os.ReadDir(path)
	if err != nil {
		reportErrorAndExit(err)
	}
	for _, entry := range entries {
		res[entry.Name()] = entry
	}
	return res
}

func reportErrorAndExit(err error) {
	fmt.Println(err)
	os.Exit(exitStatusOtherErrors)
}

func IsFileContentIdentical(file1, file2 string) bool {
	f1, err := os.Open(file1)
	if err != nil {
		reportErrorAndExit(err)
	}
	defer func(f1 *os.File) {
		err := f1.Close()
		if err != nil {
			reportErrorAndExit(err)
		}
	}(f1)

	f2, err := os.Open(file2)
	if err != nil {
		reportErrorAndExit(err)
	}
	defer func(f2 *os.File) {
		err := f2.Close()
		if err != nil {
			reportErrorAndExit(err)
		}
	}(f2)

	for {
		b1 := make([]byte, chunkSize)
		_, err1 := f1.Read(b1)

		b2 := make([]byte, chunkSize)
		_, err2 := f2.Read(b2)

		if err1 != nil || err2 != nil {
			if err1 == io.EOF && err2 == io.EOF {
				return true
			} else if err1 == io.EOF || err2 == io.EOF {
				return false
			} else {
				log.Fatal(err1, err2)
			}
		}

		if !bytes.Equal(b1, b2) {
			return false
		}
	}
}

func checkDir(sourceDir string, destDir string, notesPtr *[]string) {
	destEntries := readDirIntoMap(destDir)
	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		reportErrorAndExit(err)
	}

	for _, sourceEntry := range entries {
		destEntry, found := destEntries[sourceEntry.Name()]
		if found {
			sourcePath := filepath.Join(sourceDir, sourceEntry.Name())
			destPath := filepath.Join(destDir, destEntry.Name())
			if sourceEntry.IsDir() && destEntry.IsDir() {
				checkDir(sourcePath, destPath, notesPtr)
			} else if sourceEntry.IsDir() != destEntry.IsDir() {
				msg := fmt.Sprintf("'%s' and '%s' have different types", sourcePath, destPath)
				*notesPtr = append(*notesPtr, msg)
				fmt.Println(msg)
			} else {
				sourceInfo, err := sourceEntry.Info()
				if err != nil {
					reportErrorAndExit(err)
				}
				destInfo, err := destEntry.Info()
				if err != nil {
					reportErrorAndExit(err)
				}
				if sourceInfo.Size() != destInfo.Size() {
					msg := fmt.Sprintf("'%s' and '%s' have different sizes", sourcePath, destPath)
					*notesPtr = append(*notesPtr, msg)
					fmt.Println(msg)
				} else if !IsFileContentIdentical(sourcePath, destPath) {
					msg := fmt.Sprintf("'%s' and '%s' content differs", sourcePath, destPath)
					*notesPtr = append(*notesPtr, msg)
					fmt.Println(msg)
				}
			}
		}
	}
}

func precopyCheck(sourceDir string, destDir string) {
	var notes []string
	checkDir(sourceDir, destDir, &notes)

	if len(notes) == 0 {
		fmt.Println("Safe to copy")
	} else {
		fmt.Println("It may be unsafe")
		os.Exit(exitStatusCopyUnsafe)
		//for _, note := range notes {
		//	fmt.Println(note)
		//}
	}
}

func main() {
	helpPtr := flag.Bool("help", false, "show usage")
	flag.Parse()

	if *helpPtr || flag.Arg(0) == "" || flag.Arg(1) == "" {
		fmt.Println("Usage: precopy SRC DEST")
		fmt.Println("Exit status is only zero when merging folders is safe, so that you could chain it with rsync, e.g.:")
		fmt.Println("precopy src_folder dest_folder && rsync -ra --remove-sent-files src_folder/ dest_folder")
		fmt.Println("(Note the trailing slash with the first rsync argument")
		os.Exit(0)
	}

	sourceDir := flag.Arg(0)
	destDir := flag.Arg(1)

	fmt.Printf("Checking before copying from '%s' to '%s'\n", sourceDir, destDir)
	precopyCheck(sourceDir, destDir)
}
