package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/atotto/clipboard"
	"golang.org/x/sys/windows"
)

func getBaseDirsWin() ([]string, error) {
	ret := []string{}
	buffer := make([]uint16, 254)
	n, err := windows.GetLogicalDriveStrings(uint32(len(buffer)), &buffer[0])
	if err != nil {
		fmt.Printf("Error %v\n", err)
		return ret, err
	}

	drives := buffer[:n]
	for i := 0; i < len(drives); i += 4 {
		drive := windows.UTF16ToString(drives[i : i+4])
		ret = append(ret, drive)
	}
	return ret, nil
}

func getWorkingDir() (string, error) {
	workingDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return "", err
	}

	return workingDir, nil
}

func search(sl []string, term string) []string {
	searchList := sl
	distributedList := []string{}
	ch := make(chan string)
	for _, root := range searchList {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			distributedList = append(distributedList, root+entry.Name())
		}
	}
	var wg sync.WaitGroup
	for _, root := range distributedList {
		baseSearch := []string{root}
		wg.Add(1)
		go subSearch(baseSearch, ch, &wg, term)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()
	var results []string
	for value := range ch {
		results = append(results, value)
	}
	return results
}

func subSearch(sl []string, ch chan<- string, wg *sync.WaitGroup, term string) {
	defer wg.Done()
	searchList := sl
	tempList := []string{}
	for len(searchList) > 0 {
		for _, root := range searchList {
			tempList = []string{}
			entries, err := os.ReadDir(root)
			if err != nil {
				continue
			}

			for _, entry := range entries {
				tempList = append(tempList, root+"\\"+entry.Name())
				if strings.Contains(entry.Name(), term) {
					ch <- root + "\\" + entry.Name()
				}
			}
		}
		searchList = tempList
	}
}

func main() {
	localFlag := flag.Bool("l", false, "Local search instead of global")
	cFlag := flag.Bool("c", false, "copy a choice to clipboard after search")

	flag.Parse()

	if len(flag.Args()) < 1 {
		fmt.Println("Usage: artemis [-l for local search (working directory] [-c to enable copy to clipboard option] <search term> ")
		return
	}

	searchTerm := flag.Args()[0]

	var pathList []string
	var err error
	if *localFlag {
		path, err := getWorkingDir()
		if err != nil {
			fmt.Println("error getting working dir")
			return
		}
		pathList = []string{path}
	} else {
		pathList, err = getBaseDirsWin()
		if err != nil {
			fmt.Println("Error getting base dirs")
			return
		}
	}

	results := search(pathList, searchTerm)
	var choice int
	if len(results) > 0 {
		for _, res := range results {
			fmt.Println(res)
		}
		if *cFlag {
			fmt.Print("Enter row to copy: ")
			fmt.Scanf("%d", &choice)
			if choice <= 0 || choice > len(results) {
				fmt.Println("Cannot copy that, row doesnt exist")
				return
			}
			err := clipboard.WriteAll(results[choice-1])
			if err != nil {
				fmt.Println("copy to clipboard failed")
				return
			}
			fmt.Printf("Copied %s to clipboard \n", results[choice-1])
		}
	} else {
		fmt.Print("Nothing found")
	}
}
