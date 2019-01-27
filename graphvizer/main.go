package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
)

type universe struct {
	Jumps        []jump `json:"jumps"`
	SolarSystems []struct {
		ID     int     `json:"id"`
		Name   string  `json:"name"`
		Region string  `json:"region"`
		X      float64 `json:"x"`
		Y      float64 `json:"y"`
		Z      float64 `json:"z"`
	} `json:"solarSystems"`
}

type jump struct {
	From int `json:"from"`
	To   int `json:"to"`
}

func main() {
	fb, err := ioutil.ReadFile("tldr.json")
	if err != nil {
		log.Fatalf("Failed to read tldr.json, run bird-parser first")
	}

	var output map[string]int

	err = json.Unmarshal(fb, &output)
	if err != nil {
		log.Fatalf("Failed to parse tldr.json %s", err.Error())
	}

	// b, err := ioutil.ReadFile("../universe-pretty.json")
	b, err := ioutil.ReadFile("../tfl.json")
	if err != nil {
		log.Fatalf("Unable to read JSON data file, cannot build configs without this %s", err.Error())
	}

	NewEden := universe{}
	err = json.Unmarshal(b, &NewEden)
	if err != nil {
		log.Fatalf("Unable to decode data: %s", err.Error())
	}

	os.Stdout.Write([]byte("digraph world {"))
	os.Stdout.Write([]byte("\noverlap = compress;sep=2;scale=.25;\n"))
	// 	os.Stdout.Write([]byte(`
	// {
	// 	rankdir=TB;
	// }`))

	UsedSystems := ListedPaths(output)
	NameOfSystems := NamesPls(NewEden)
	SubRegions := make(map[string][]int)
	for k, v := range NewEden.SolarSystems {
		// SubRegions[v.Region] += 130
		// TotalRAM += 130
		arr := SubRegions[v.Region]
		if arr == nil {
			arr = make([]int, 0)
		}
		if UsedSystems[k+1] {
			arr = append(arr, k+1)
			SubRegions[v.Region] = arr
		}
	}

	i := 0
	for region, items := range SubRegions {
		s := ""
		for _, v := range items {

			s += fmt.Sprintf("\"%s\";", NameOfSystems[v])

		}
		os.Stdout.Write([]byte(fmt.Sprintf(`
		subgraph cluster_%d {
			ranksep=3;
			style="filled";
			color=lightblue;
			fontname="DejaVu";
			node [style=filled,color="#005ea5",fillcolor=white,shape=rectangle,penwidth=3,fontname="DejaVu",width=.25,height=.375,fontsize=9];
			%s
			label = "%s";
		}`, i, s, region)))
		i++
	}

	for k, _ := range output {
		os.Stdout.Write([]byte(fmt.Sprintf("%s;\n", NameRewrite(NameOfSystems, k))))
	}
	os.Stdout.Write([]byte("}"))
}

func NameRewrite(in map[int]string, str string) string {
	// 1->3
	s := strings.Split(str, "-")
	s[1] = strings.Trim(s[1], ">")

	out := ""

	i, _ := strconv.ParseInt(s[0], 10, 64)
	out = "\"" + in[int(i)] + "\""
	out += " -> "
	i, _ = strconv.ParseInt(s[1], 10, 64)
	out += "\"" + in[int(i)] + "\""
	return out
}

func NamesPls(in universe) map[int]string {
	o := make(map[int]string)
	i := 1
	for _, v := range in.SolarSystems {
		o[v.ID] = v.Name
		i++
	}
	return o
}

func ListedPaths(in map[string]int) map[int]bool {
	o := make(map[int]bool)

	for v, _ := range in {
		s := strings.Split(v, "-")
		s[1] = strings.Trim(s[1], ">")

		i, _ := strconv.ParseInt(s[0], 10, 64)
		o[int(i)] = true
		i, _ = strconv.ParseInt(s[1], 10, 64)
		o[int(i)] = true
	}

	return o
}
