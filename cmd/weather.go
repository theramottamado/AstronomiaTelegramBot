package cmd

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

func GetWeather(city string) (weather string) {
	token := os.Getenv("API_TOKEN")
	resp, err := http.Get("http://api.openweathermap.org/data/2.5/weather?q=" + city + "&appid=" + token)
	if err != nil {
		log.Println(err)
		return "Network error!"
	}
	defer resp.Body.Close()
	defer func() {
		if err := recover(); err != nil {
			weather = "oops!"
		}
	}()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return "oops!"
	}

	var result map[string]interface{}
	json.Unmarshal([]byte(body), &result)

	if resp.StatusCode != 200 {
		msg := result["message"]
		return msg.(string)
	}

	weatherMap := result["weather"].([]interface{})
	weatherArr := weatherMap[0].(map[string]interface{})
	weather = weatherArr["description"].(string)
	return weather
}
