package main

import (
	"fmt"
	"io"
	"os"
	"strings"
)

func main() {
	out := os.Stdout
	if !(len(os.Args) == 2 || len(os.Args) == 3) {
		panic("usage go run main.go . [-f]")
	}
	path := os.Args[1]
	printFiles := len(os.Args) == 3 && os.Args[2] == "-f"
	err := dirTree(out, path, printFiles)
	if err != nil {
		panic(err.Error())
	}
}

func dirTree(out io.Writer, path string, printFiles bool) error {
	err := dirTree2(out, path, printFiles, "")
	return err
}

func dirTree2(out io.Writer, path string, printFiles bool, prefixPrev string) error {
	files, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	var printItem []os.DirEntry

	if printFiles {
		printItem = files
	} else {
		for _, file := range files {
			if file.IsDir() {
				printItem = append(printItem, file)
			}
		}
	}

	for i, value := range printItem {
		var prefix, nextPrefix string

		if i != len(printItem)-1 {
			prefix = prefixPrev + "├───"
			nextPrefix = prefixPrev + "│	"
		} else {
			prefix = prefixPrev + "└───"
			nextPrefix = prefixPrev + "\t"
		}

		fmt.Fprint(out, prefix)

		if value.IsDir() {
			fmt.Fprintln(out, value.Name())
			dirTree2(out, path+string(os.PathSeparator)+value.Name(), printFiles, nextPrefix)
		} else {
			fileInfo, err := os.Stat(path + string(os.PathSeparator) + value.Name())
			if err != nil {
				return err
			}

			var sb strings.Builder = strings.Builder{}
			sb.WriteString(fileInfo.Name())
			if fileInfo.Size() == 0 {
				sb.WriteString(" (empty)")
			} else {
				sb.WriteString(fmt.Sprintf(" (%db)", fileInfo.Size()))
			}
			fmt.Fprintln(out, sb.String())
		}
	}

	return nil
}
