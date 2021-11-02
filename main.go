package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"sort"
	"time"
)

type ServeryDataType int64

const (
	food ServeryDataType = iota
	day
	mealTime
	Servery
)

type ServeryData struct {
	DataType ServeryDataType
	Text     string
	Position int
	Servery  string
	Time     string
}

func (data ServeryData) toJson() string {
	b, _ := json.Marshal(data)
	c := fmt.Sprintf("%q", b)
	return c
}

func getMatchesWithIndex(body []byte, myregex *regexp.Regexp) ([][]byte, [][]int) {
	matched := myregex.FindAll(body, -1)
	matchedIdx := myregex.FindAllIndex(body, -1)
	return matched, matchedIdx
}

func getServeryData(Servery string) []ServeryData {

	url := fmt.Sprintf("https://websvc-aws.rice.edu:8443/static-files/dining-assets/%s-Menu-Full-Week.js", Servery)

	resp, _ := http.Get(url)
	body, _ := io.ReadAll(resp.Body)

	data := make([]ServeryData, 0)

	timeRegex, _ := regexp.Compile(`class=\\"meal-time meal-time-[^\\]*`)
	timeMatched, timeMatchedIdx := getMatchesWithIndex(body, timeRegex)

	dinnerPos := 0

	for idx, slice := range timeMatched {

		if fmt.Sprintf("%q", slice[28:]) == "\"dinner\"" {
			dinnerPos = timeMatchedIdx[idx][0]
		}

		timeStr := fmt.Sprintf("%q", slice[28:])
		data = append(data, ServeryData{
			DataType: mealTime,
			Text:     timeStr,
			Position: timeMatchedIdx[idx][0],
			Servery:  Servery,
			Time:     timeStr[1 : len(timeStr)-1]})
	}

	foodsRegex, _ := regexp.Compile(`class=\\"mitem\\"\\u003E[^\\]*`)
	foodsMatched, foodsMatchedIdx := getMatchesWithIndex(body, foodsRegex)

	for idx, slice := range foodsMatched {
		var mealTime string
		if foodsMatchedIdx[idx][0] < dinnerPos {
			mealTime = "lunch"
		} else {
			mealTime = "dinner"
		}

		data = append(data, ServeryData{
			DataType: food,
			Text:     fmt.Sprintf("%q", slice[21:]),
			Position: foodsMatchedIdx[idx][0],
			Servery:  Servery,
			Time:     mealTime})
	}

	daysRegex, _ := regexp.Compile(`style=\\"background:#212d64;\\"\\u003E[^\\]*`)
	daysMatched, daysMatchedIdx := getMatchesWithIndex(body, daysRegex)

	for idx, slice := range daysMatched {
		var mealTime string
		if daysMatchedIdx[idx][0] < dinnerPos {
			mealTime = "lunch"
		} else {
			mealTime = "dinner"
		}

		data = append(data, ServeryData{
			DataType: day,
			Text:     fmt.Sprintf("%q", slice[35:len(slice)-1]),
			Position: daysMatchedIdx[idx][0],
			Servery:  Servery,
			Time:     mealTime})
	}

	sort.Slice(data, func(i, j int) bool {
		return data[i].Position < data[j].Position
	})

	return data
}

func getAllServeryData() []ServeryData {
	serveries := []string{"Baker-Kitchen", "North-Servery", "West-Servery", "South-Servery", "Seibel-Servery"}

	data := make([]ServeryData, 0)
	data = append(data, ServeryData{
		DataType: food,
		Text:     fmt.Sprintf("Last Fetched %s", time.Now()),
		Position: 0,
		Servery:  "N/A",
		Time:     "N/A"})
	for _, v := range serveries {
		data = append(data, ServeryData{
			DataType: Servery,
			Text:     v,
			Position: 0,
			Servery:  v,
			Time:     "N/A"})
		data = append(data, getServeryData(v)...)
	}

	return data
}

func main() {

	data := getAllServeryData()

	go func() {
		c := time.Tick(time.Minute)
		for next := range c {
			fmt.Print("Tick", next)
			data = getAllServeryData()
		}
	}()

	/*for _, slice := range data {
		//fmt.Printf("Type: %d\n", slice.DataType)
		//fmt.Printf("Text: %s\n", slice.Text)
		//fmt.Printf("Position: %d\n", slice.Position)
		switch slice.DataType {
		case day:
			fmt.Printf("\n---%s---\n", slice.Text)
		case time:
			fmt.Printf("\n\n---------------------\n-------%s-------\n---------------------\n", slice.Text)
		case food:
			fmt.Printf("%s\n", slice.Text)
		case Servery:
			fmt.Printf("\n\n%s\n\n\n", slice.Text)
		}
	}*/

	fileServer := http.FileServer(http.Dir("./src"))
	http.Handle("/", fileServer)

	http.HandleFunc("/data", func(w http.ResponseWriter, r *http.Request) {
		jsonSlice := make([]string, 0)
		for _, slice := range data {
			jsonSlice = append(jsonSlice, slice.toJson())
		}
		w.Header().Set("Content-type", "application/json")
		b, _ := json.Marshal(struct{ Jsons []string }{Jsons: jsonSlice})
		fmt.Fprintf(w, "%q", b)
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
	//log.Fatal(http.ListenAndServeTLS(":443", "./https/server.crt", "./https/server.key", nil))

}
