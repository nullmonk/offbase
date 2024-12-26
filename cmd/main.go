package main

import (
	"fmt"
	"offbase"
	"os"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: offbase <url> <destination>")
		os.Exit(1)
	}

	url := os.Args[1]
	destination := os.Args[2]

	s, err := offbase.NewScraper(url, destination)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err := s.Scrape(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	_, err = s.GetRootDirectory()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	for _, f := range s.GetFiles() {
		fmt.Println(f.FullPath())
	}
}
