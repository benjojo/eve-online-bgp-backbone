package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
)

type universe struct {
	Jumps        []jump   `json:"jumps"`
	SolarSystems []system `json:"solarSystems"`
}

type system struct {
	ID     int     `json:"id"`
	Name   string  `json:"name"`
	Region string  `json:"region"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Z      float64 `json:"z"`
}

type jump struct {
	From    int `json:"from"`
	To      int `json:"to"`
	Latency int `json:"latency"`
}

func main() {
	//
	f, err := os.Open("./TFLMap.csv")
	if err != nil {
		log.Fatalf("Unable to open ./TFLMap.csv, please check the repo for that file. : %s", err.Error())
	}

	csvreader := csv.NewReader(f)

	TFLUniverse := universe{
		Jumps:        make([]jump, 0),
		SolarSystems: make([]system, 0),
	}

	AlreadyExistsMap := make(map[string]int)

	for {
		record, err := csvreader.Read()
		if err == io.EOF {
			// reading done
			break
		}

		if err != nil {
			log.Printf("CSV error: %v", err.Error())
			continue
		}

		// Line
		// Station from (A)
		// Station to (B)
		// Distance (Kms)
		// Un-impeded Running Time (Mins)
		// AM peak (0700-1000) Running Time (Mins)
		// Inter peak (1000 - 1600) Running time (mins)

		StationA, StationB := SaneName(record[1]), SaneName(record[2])
		SAID := MakeStationIfNotExists(StationA, &TFLUniverse, AlreadyExistsMap)
		SBID := MakeStationIfNotExists(StationB, &TFLUniverse, AlreadyExistsMap)

		distance, _ := strconv.ParseFloat(record[4], 32)

		// now make the jump?
		j := jump{
			From:    SAID,
			To:      SBID,
			Latency: int(distance * 100),
		}

		TFLUniverse.Jumps = append(TFLUniverse.Jumps, j)
	}

	b, _ := json.Marshal(TFLUniverse)
	ioutil.WriteFile("tfl.json", b, 0777)
}

var SystemsInExistance = 1

func MakeStationIfNotExists(in string, uni *universe, AlreadyExistsMap map[string]int) int {
	if AlreadyExistsMap[in] != 0 {
		return AlreadyExistsMap[in]
	}

	// now we have to make it...

	s := system{
		Name:   in,
		ID:     SystemsInExistance + 1,
		Region: "tfl",
	}

	SystemsInExistance++
	AlreadyExistsMap[in] = SystemsInExistance

	uni.SolarSystems = append(uni.SolarSystems, s)
	return SystemsInExistance
}

func SaneName(in string) string {
	if strings.Contains(in, " ") {
		bits := strings.Split(in, " ")
		for k, v := range bits {
			t := strings.ToLower(v)
			u := strings.ToUpper(v)
			bits[k] = fmt.Sprintf("%s%s", string([]byte(u)[0]), string([]byte(t)[1:]))
		}

		return strings.Join(bits, "-")
	}

	t := strings.ToLower(in)
	u := strings.ToUpper(in)
	return fmt.Sprintf("%s%s", string([]byte(u)[0]), string([]byte(t)[1:]))

}
