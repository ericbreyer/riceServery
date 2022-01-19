package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
)

type DataBlockType int64

const (
	food DataBlockType = iota
	day
	mealTime
	servery
)

type DataBlock struct {
	DataType DataBlockType
	Text     string
	Position int
}

type serveryGroup struct {
	Name           string
	MealTimeGroups []mealTimeGroup
}

type mealTimeGroup struct {
	Name          string
	MealDayGroups []mealDayGroup
}

type mealDayGroup struct {
	Name   string
	Meals  []meal
	Id     string
	Rating int
}

type meal struct {
	Name   string
	Rating int
}

type averageRating struct {
	AverageRating   float32
	totalRating     int
	numberOfRatings int64
}

func (s *averageRating) addRating(rating int) {
	s.numberOfRatings++
	s.totalRating += rating
	s.AverageRating = float32(s.totalRating) / float32(s.numberOfRatings)
}

var idToRating map[string]averageRating

func getMatchesWithIndex(body []byte, myregex *regexp.Regexp) ([][]byte, [][]int) {
	matched := myregex.FindAll(body, -1)
	matchedIdx := myregex.FindAllIndex(body, -1)
	return matched, matchedIdx
}

func getServeryData(Servery string) (serveryGroup, error) {

	url := fmt.Sprintf("https://websvc-aws.rice.edu:8443/static-files/dining-assets/%s-Menu-Full-Week.js", Servery)

	resp, err := http.Get(url)

	if err != nil {
		fmt.Printf("%v", err)
		return serveryGroup{}, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		fmt.Printf("%v", err)
		return serveryGroup{}, err
	}

	rawData := make([]DataBlock, 0)

	timeRegex, _ := regexp.Compile(`class=\\"meal-time meal-time-[^\\]*`)
	timeMatched, timeMatchedIdx := getMatchesWithIndex(body, timeRegex)

	for idx, slice := range timeMatched {

		timeStr := fmt.Sprintf("%s", slice[28:])
		rawData = append(rawData, DataBlock{
			DataType: mealTime,
			Text:     timeStr,
			Position: timeMatchedIdx[idx][0]})
	}

	foodsRegex, _ := regexp.Compile(`class=\\"mitem\\"\\u003E[^\\]*`)
	foodsMatched, foodsMatchedIdx := getMatchesWithIndex(body, foodsRegex)

	for idx, slice := range foodsMatched {

		rawData = append(rawData, DataBlock{
			DataType: food,
			Text:     strings.TrimSpace(fmt.Sprintf("%s", slice[21:])),
			Position: foodsMatchedIdx[idx][0]})
	}

	daysRegex, _ := regexp.Compile(`style=\\"background:#212d64;\\"\\u003E[^\\]*`)
	daysMatched, daysMatchedIdx := getMatchesWithIndex(body, daysRegex)

	for idx, slice := range daysMatched {

		rawData = append(rawData, DataBlock{
			DataType: day,
			Text:     fmt.Sprintf("%s", slice[35:len(slice)-1]),
			Position: daysMatchedIdx[idx][0]})
	}

	sort.Slice(rawData, func(i, j int) bool {
		return rawData[i].Position < rawData[j].Position
	})

	data := serveryGroup{Name: Servery}
	var currentMealTimeBlock mealTimeGroup
	var currentMealDayBlock mealDayGroup
	for _, block := range rawData {
		switch block.DataType {
		case servery:
			continue
		case mealTime:
			if currentMealTimeBlock.Name != "" {
				currentMealTimeBlock.MealDayGroups = append(currentMealTimeBlock.MealDayGroups, currentMealDayBlock)
				currentMealDayBlock = mealDayGroup{}
				data.MealTimeGroups = append(data.MealTimeGroups, currentMealTimeBlock)
			}
			currentMealTimeBlock = mealTimeGroup{Name: block.Text}
		case day:
			if currentMealDayBlock.Name != "" {
				currentMealTimeBlock.MealDayGroups = append(currentMealTimeBlock.MealDayGroups, currentMealDayBlock)
			}

			year, week := time.Now().ISOWeek()
			currentMealDayBlock = mealDayGroup{Name: block.Text, Id: fmt.Sprintf("%d%s%d/%d", block.Position, Servery, week, year)}
		case food:
			rating := 0

			if val, ok := idToRating[block.Text]; ok {
				rating = int(math.Round(float64(val.AverageRating)))
			}

			if block.Text != "CLOSED" && block.Text != "Closed" {
				currentMealDayBlock.Meals = append(currentMealDayBlock.Meals, meal{Name: block.Text, Rating: rating})
			}

		}
	}
	currentMealTimeBlock.MealDayGroups = append(currentMealTimeBlock.MealDayGroups, currentMealDayBlock)
	data.MealTimeGroups = append(data.MealTimeGroups, currentMealTimeBlock)

	//dataJson, _ := json.MarshalIndent(data, "", "  ")
	//fmt.Printf("%s", dataJson)

	return data, nil
}

func getAllServeryData() []serveryGroup {
	fmt.Print("Tick! ", time.Now(), "\n")

	serveries := []string{"Baker-Kitchen", "North-Servery", "West-Servery", "South-Servery", "Seibel-Servery"}

	data := make([]serveryGroup, 0)
	//data = append(data, serveryGroup{
	//	Name: fmt.Sprintf("Last Fetched %s", time.Now())})

	for _, v := range serveries {
		serveryData, err := getServeryData(v)
		for err != nil {
			println("\n----\nahh\n----\n", err)
			serveryData, err = getServeryData(v)
		}
		data = append(data, serveryData)
	}

	return data
}

func main() {

	idToRating = make(map[string]averageRating)

	data := getAllServeryData()

	go func() {

		c := time.Tick(time.Minute)
		for range c {
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
		for _, servery := range data {
			serveryJson, _ := json.Marshal(servery)
			jsonSlice = append(jsonSlice, fmt.Sprintf("%s", serveryJson))
		}
		w.Header().Set("Content-type", "application/json")
		b, _ := json.Marshal(struct{ Jsons []string }{Jsons: jsonSlice})
		fmt.Fprintf(w, "%s", b)
	})

	http.HandleFunc("/updateRating", func(w http.ResponseWriter, r *http.Request) {
		body, _ := ioutil.ReadAll(r.Body)
		structure := &meal{}
		_ = json.Unmarshal(body, structure)

		if val, ok := idToRating[(*structure).Name]; ok {
			val.addRating((*structure).Rating)
			idToRating[(*structure).Name] = val
		} else {
			rating := averageRating{}
			rating.addRating((*structure).Rating)
			idToRating[(*structure).Name] = rating
		}

		fmt.Print("Update ", (*structure).Name, " ")
		data = getAllServeryData()

		fmt.Fprintf(w, "All Good")
	})
	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./src/assets/crab.ico")
	})

	port := os.Getenv("PORT")

	if port == "" {
		fmt.Print("Server started on port 8080")
		log.Fatal(http.ListenAndServe(":8080", nil))
	} else {
		fmt.Print("Server started on port " + port)
		log.Fatal(http.ListenAndServe(":"+port, nil))
	}
	//log.Fatal(http.ListenAndServeTLS(":443", "./https/server.crt", "./https/server.key", nil))

}
