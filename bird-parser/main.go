package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
)

func main() {
	files, err := ioutil.ReadDir(".")
	if err != nil {
		log.Fatalf("Unable to read current directory. %s", err.Error())
	}

	pathScores := make(map[string]int)

	for _, v := range files {

		bits := strings.Split(v.Name(), ".")
		if len(bits) != 2 {
			continue
		}

		_, err := strconv.ParseInt(bits[0], 10, 64)
		if err != nil {
			continue
		}

		_, err = strconv.ParseInt(bits[1], 10, 64)
		if err != nil {
			continue
		}

		// Ok, so it's a file we want
		// let's read it!

		f, err := os.Open(v.Name())
		if err != nil {
			fmt.Print("!")
			continue
		}
		fmt.Print(".")

		bio := bufio.NewReader(f)
		for {
			line, overflow, err := bio.ReadLine()

			if err != nil {
				break
			}

			if overflow {
				continue
			}

			if !strings.Contains(string(line), "BGP.as_path") {
				continue
			}

			linebits := strings.Split(string(line), " ")
			pathsPresent := false
			for k, bit := range linebits {
				if !pathsPresent {
					if strings.HasSuffix(bit, ":") {
						pathsPresent = true
						continue
					}
				} else {
					if len(linebits)-1 == k {
						// avoid seeking beyond the array
						continue
					}

					backASN, _ := strconv.ParseInt(bit, 10, 64)
					fwdASN, _ := strconv.ParseInt(linebits[k+1], 10, 64)

					if fwdASN == 0 || backASN == 0 {
						continue
					}

					pathScores[sortRight(fwdASN, backASN)]++
				}
			}
		}
	}

	// Paths computed, let's split out some scores

	os.Stdout.Write([]byte("digraph world {"))
	for k, _ := range pathScores {
		os.Stdout.Write([]byte(fmt.Sprintf("%s;\n", k)))
	}
	os.Stdout.Write([]byte("}"))

	b, _ := json.Marshal(pathScores)
	ioutil.WriteFile("tldr.json", b, 0777)
}

func sortRight(a, b int64) string {
	if a > b {
		return fmt.Sprintf("%d->%d", a, b)
	}

	return fmt.Sprintf("%d->%d", b, a)
}
