package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/as3richa/lthea9"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Printf("usage: %s <filename>\n", os.Args[0])
		os.Exit(1)
	}

	fmt.Println("building index...")
	index := buildIndex(os.Args[1])
	fmt.Println("... done")

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("> ")
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if line == "" || err == io.EOF {
			break
		}
		index.Query(strings.TrimRight(line, "\n"), 50, func(res lthea9.QueryResult) {
			fmt.Println(res.Str)
			s := ""
			for _, pos := range res.Pos {
				for len(s) < int(pos) {
					s += " "
				}
				s += "*"
			}
			fmt.Println(s)
		})
	}
}

func buildIndex(filename string) lthea9.SubseqIndex {
	file, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	reader := bufio.NewReader(file)

	builder := lthea9.SubseqIndexBuilder{}

	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err == io.EOF || line == "" {
			break
		}
		builder.Insert(strings.TrimRight(line, "\n"))
	}

	return builder.Build()
}
