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
	"sync"
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

	//url := fmt.Sprintf("https://websvc-aws.rice.edu:8443/static-files/dining-assets/%s-Menu-Full-Week.js", Servery)
	url := fmt.Sprintf("https://web-api3.rice.edu/static/%s-menu-new.js", Servery)
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
	fmt.Print("Data Update Tick! at: ", time.Now(), "\n")

	serveries := []string{"baker-kitchen", "north-servery", "west-servery", "south-servery", "seibel-servery"}

	data := make([]serveryGroup, 0)

	w := sync.WaitGroup{}
	w.Add(len(serveries))
	for _, servery := range serveries {
		//concurrency is about twice as fast and its cool!
		go func(v string) {
			serveryData, err := getServeryData(v)
			for err != nil {
				println("\n----\nahh\n----\n", err)
				serveryData, err = getServeryData(v)
			}
			data = append(data, serveryData)
			w.Done()
		}(servery)
	}

	w.Wait()
	return data
}

func getDateString() (day string) {
	i, j, k := time.Now().Date()
	day = fmt.Sprintf("%d%s%d", i, j.String(), k)
	return
}

func addToJSON() {
	temp, _ := os.ReadFile("./db.json")
	myjson := struct {
		Data []struct {
			Date      string
			Serveries []serveryGroup
		}
	}{}
	json.Unmarshal(temp, &myjson)

	day := getDateString()

	myjsoncopy := myjson
	for i, obj := range myjson.Data {
		if obj.Date == day {
			myjsoncopy.Data = append(myjson.Data[:i], myjson.Data[i+1:]...)
		}
	}
	myjson = myjsoncopy

	myjson.Data = append(myjson.Data, struct {
		Date      string
		Serveries []serveryGroup
	}{day, getAllServeryData()})
	myJsonString, _ := json.MarshalIndent(myjson, "", "  ")
	os.WriteFile("./db.json", myJsonString, 0666)
}

func main() {

	idToRating = make(map[string]averageRating)

	data := getAllServeryData()
	addToJSON()

	//update data every minute concurrently!
	go func() {
		c := time.Tick(time.Minute)
		for range c {
			data = getAllServeryData()
			addToJSON()
		}
	}()

	fileServer := http.FileServer(http.Dir("./src"))
	http.Handle("/", fileServer)

	http.HandleFunc("/data", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			return
		}
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
		if r.Method != "POST" {
			return
		}

		body, _ := ioutil.ReadAll(r.Body)
		mealToUpdate := &meal{}
		_ = json.Unmarshal(body, mealToUpdate)

		if val, ok := idToRating[(*mealToUpdate).Name]; ok {
			val.addRating((*mealToUpdate).Rating)
			idToRating[(*mealToUpdate).Name] = val
		} else {
			rating := averageRating{}
			rating.addRating((*mealToUpdate).Rating)
			idToRating[(*mealToUpdate).Name] = rating
		}

		fmt.Print("Update ", (*mealToUpdate).Name, " ")
		data = getAllServeryData()

		fmt.Fprintf(w, "All Good")
	})

	port := os.Getenv("PORT")

	if port == "" {
		fmt.Print("Server started on port 8080")
		log.Fatal(http.ListenAndServe(":8080", nil))
	} else {
		fmt.Print("Server started on port " + port)
		log.Fatal(http.ListenAndServe(":"+port, nil))
	}
}
