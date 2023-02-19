package main

import (
	"compress/gzip"
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

	"github.com/dsnet/compress/brotli"
)

type DataBlockType int64

const (
	food DataBlockType = iota
	day
	mealTime
	servery
	allergtre
)

type DataBlock struct {
	DataType DataBlockType
	Text     string
	Position int
}

type serveryGroup struct {
	Name           string
	MealTimeGroups []mealTimeGroup
	Chef           chefName
}

type chefName struct {
	Name string
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
	Name     string
	Rating   int
	Alergies []allergy
}

type allergy struct {
	Name string
}

var alergyToID map[string]allergy = map[string]allergy{
	"eggs":       {"E"},
	"fish":       {"F"},
	"gluten":     {"G"},
	"milk":       {"M"},
	"peanuts":    {"P"},
	"shellfish":  {"Sh"},
	"soy":        {"So"},
	"tree-nuts":  {"N"},
	"vegan":      {"Veg"},
	"vegetarian": {"V"},
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
	// url0 := "https://dining.rice.edu/baker-college-kitchen/full-week-menu"
	// req0, _ := http.NewRequest("GET", url0, nil)
	// resp0, err := http.DefaultClient.Do(req0)
	// print(len(resp0.Cookies()))
	// body0, err := io.ReadAll(resp0.Body)
	// fmt.Printf("%s", body0)

	//url := fmt.Sprintf("https://websvc-aws.rice.edu:8443/static-files/dining-assets/%s-Menu-Full-Week.js", Servery)
	url := fmt.Sprintf("https://staticws.b-cdn.net/dining/%s-menu-full-week-new.js", Servery)
	//url := fmt.Sprintf("https://web-api3.rice.edu/static/%s-menu-full-week-new.js", Servery)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Host", "staticws.b-cdn.net")
	req.Header.Add("Accept", "	text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9")
	req.Header.Add("Cl", "gzip, deflate, br")
	req.Header.Add("accept-encoding", "gzip, deflate, br")
	req.Header.Add("accept-language", "en-US,en;q=0.9")
	req.Header.Add("cache-control", "max-age=0")
	req.Header.Add("sec-fetch-mode", "navigate")
	req.Header.Add("sec-fetch-site", "none")
	req.Header.Add("sec-fetch-user", "?1")
	req.Header.Add("upgrade-insecure-requests", "1")
	req.Header.Add("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/98.0.4758.87 Safari/537.36")
	req.Header.Add("Connection", "keep-alive")
	req.Header.Add("Sec-Fetch-Site", "cross-site")

	req.AddCookie(&http.Cookie{Name: "NID", Value: "RUxmt2dUKb6R_NlYrONl5C9Dcj0y0BzVXgo7yC_kHqA_zzHsM1NGUrGXvZ5_FBYchZ5u7LyHfwOBSfvYdguZOiukqdPYFgLPCY2DU4l5pqdy8tNBd0GklQT7FCci6AypX_DQsBemZj2mOsMz70zGe5zSwp_8m1_Kod27EiSGvqTPsvQz8s_5e_Jc0LpH"})
	req.AddCookie(&http.Cookie{Name: "__Secure-3PAPISID", Value: "8xq3QzorcCtMrhZs/AH87b3RT32JSYRpGt"})
	req.AddCookie(&http.Cookie{Name: "__Secure-3PSID", Value: "NwjqU1Nr8BRMjtiUpdSj0FGOWiFbx27YMt57xKk7Efx3kCTWSmQRnA-kGQ5EWYTd2y4JJg."})
	req.AddCookie(&http.Cookie{Name: "__Secure-3PSIDCC", Value: "AEf-XMQ4hLqlQQKIUNNcvWy6KkdJ1mEbTmrk4SxVKqV7cloczy7F5TrJD8oBiAmkGC8sO5iRMw"})
	resp, err := http.DefaultClient.Do(req)
	//resp, err := http.Get(url)
	//fmt.Printf("%v\n", resp.Header)
	if err != nil {
		fmt.Printf("%v", err)
		return serveryGroup{}, err
	}
	var reader io.ReadCloser

	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		reader, _ = gzip.NewReader(resp.Body)
		defer reader.Close()
	case "br":
		reader, _ = brotli.NewReader(resp.Body, &brotli.ReaderConfig{})
	default:
		reader = resp.Body
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(reader)

	if err != nil {
		fmt.Printf("%v", err)
		return serveryGroup{}, err
	}

	rawData := make([]DataBlock, 0)

	chefRegex, _ := regexp.Compile(`<span class=\\"chef-name\\">[^<]*`)
	chefMatched, _ := getMatchesWithIndex(body, chefRegex)
	chef := chefName{Name: string(chefMatched[0][26:])}

	timeRegex, _ := regexp.Compile(`<span class=\\"meal-time[^>]*>[^<]*`)
	timeMatched, timeMatchedIdx := getMatchesWithIndex(body, timeRegex)
	for idx, slice := range timeMatched {

		temp, _ := regexp.Compile(`Lunch|Dinner`)
		matched := temp.Find(slice)
		if matched == nil {
			fmt.Printf("%s", slice)
			continue
		}

		rawData = append(rawData, DataBlock{
			DataType: mealTime,
			Text:     string(matched),
			Position: timeMatchedIdx[idx][0]})
	}

	foodsRegex, _ := regexp.Compile(`<div class=\\"mitem\\">[^<]*`)
	foodsMatched, foodsMatchedIdx := getMatchesWithIndex(body, foodsRegex)

	for idx, slice := range foodsMatched {

		rawData = append(rawData, DataBlock{
			DataType: food,
			Text:     strings.TrimSpace(fmt.Sprintf("%s", slice[21:])),
			Position: foodsMatchedIdx[idx][0]})
	}

	allergiesRegex, _ := regexp.Compile(`<div class=\\"icons icon-only icons-([^\\]*)`)
	allergiesMatched, allergiesMatchedIdx := getMatchesWithIndex(body, allergiesRegex)

	for idx, slice := range allergiesMatched {

		rawData = append(rawData, DataBlock{
			DataType: allergtre,
			Text:     strings.TrimSpace(fmt.Sprintf("%s", slice[35:])),
			Position: allergiesMatchedIdx[idx][0]})
	}

	dayTimeRegex, _ := regexp.Compile(`background:#212d64;\\">[^<]*`)
	dayTimeMatched, dayTimeMatchedIdx := getMatchesWithIndex(body, dayTimeRegex)

	for idx, slice := range dayTimeMatched {

		temp, _ := regexp.Compile(`Monday|Tuesday|Wednesday|Thursday|Friday|Saturday|Sunday`)
		matched := temp.Find(slice)
		if matched == nil {
			fmt.Printf("%s", slice)
			continue
		}

		rawData = append(rawData, DataBlock{
			DataType: day,
			Text:     fmt.Sprintf("%s", matched),
			Position: dayTimeMatchedIdx[idx][0]})
	}

	sort.Slice(rawData, func(i, j int) bool {
		return rawData[i].Position < rawData[j].Position
	})

	data := serveryGroup{Name: Servery, Chef: chef}
	var currentMealTimeBlock mealTimeGroup
	var currentMealDayBlock mealDayGroup
	var currentFoodBlock meal
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
				currentMealDayBlock.Meals = append(currentMealDayBlock.Meals, currentFoodBlock)
				currentFoodBlock = meal{}
				currentMealTimeBlock.MealDayGroups = append(currentMealTimeBlock.MealDayGroups, currentMealDayBlock)
			}

			year, week := time.Now().ISOWeek()
			currentMealDayBlock = mealDayGroup{Name: block.Text, Id: fmt.Sprintf("%d%s%d/%d", block.Position, Servery, week, year)}
		case food:
			if currentFoodBlock.Name != "" {
				currentMealDayBlock.Meals = append(currentMealDayBlock.Meals, currentFoodBlock)
			}
			rating := 0

			if val, ok := idToRating[block.Text]; ok {
				rating = int(math.Round(float64(val.AverageRating)))
			}

			if block.Text != "CLOSED" && block.Text != "Closed" {
				currentFoodBlock = meal{Name: block.Text, Rating: rating}
			}
		case allergtre:
			currentFoodBlock.Alergies = append(currentFoodBlock.Alergies, alergyToID[block.Text])
		}
	}
	currentMealDayBlock.Meals = append(currentMealDayBlock.Meals, currentFoodBlock)
	currentMealTimeBlock.MealDayGroups = append(currentMealTimeBlock.MealDayGroups, currentMealDayBlock)
	data.MealTimeGroups = append(data.MealTimeGroups, currentMealTimeBlock)

	// dataJson, _ := json.MarshalIndent(data, "", "  ")
	// fmt.Printf("%s", dataJson)

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

	//update data every hour concurrently!
	go func() {
		c := time.Tick(time.Hour)
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
		jsonSlice := make([]serveryGroup, 0)
		jsonSlice = append(jsonSlice, data...)

		w.Header().Set("Content-type", "application/json")
		b, _ := json.Marshal(struct{ Jsons []serveryGroup }{Jsons: jsonSlice})
		fmt.Fprintf(w, "%s", b)
		// fmt.Printf("%s", b)
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

	// this is for heroku
	port := os.Getenv("PORT")

	if port == "" {
		fmt.Print("Server started on port 8080")
		log.Fatal(http.ListenAndServe(":8080", nil))
	} else {
		fmt.Print("Server started on port " + port)
		log.Fatal(http.ListenAndServe(":"+port, nil))
	}
}
