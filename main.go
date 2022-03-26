package main

import (
	"fmt"
	"githubscanner/scanner"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("account is not specified")
		os.Exit(1)
	}

	items, err := scanner.GetDefaultScanner().ScanRepositories(os.Args[1])
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	for _, item := range items {
		fmt.Println(item.Repository.FullName)
		for _, release := range item.Releases {
			fmt.Println(release.Name)
		}
		fmt.Println()
	}
}
