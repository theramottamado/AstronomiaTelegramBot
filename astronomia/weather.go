package astronomia

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

type Address struct {
	FormattedAddress string
	Latitude         string
	Longitude        string
}

func (a *Address) SetFormattedAddress(address string) {
	a.FormattedAddress = address
}

func (a *Address) SetLatitude(lat string) {
	a.Latitude = lat
}

func (a *Address) SetLongitude(lon string) {
	a.Longitude = lon
}

func GetWeather(firstName, lastName, address string) (weather string, err error) {
	token := os.Getenv("API_TOKEN")
	location, err := getLatLon(address)
	if err != nil {
		return "", err
	}
	formattedAddress, lat, lon := location.FormattedAddress, location.Latitude, location.Longitude
	resp, err := http.Get("http://api.openweathermap.org/data/2.5/weather?lat=" + lat + "&lon=" + lon + "&appid=" + token + "&units=metric")
	if err != nil {
		err = errors.New("weather not found for this location")
		return "Network error!", err
	}
	defer resp.Body.Close()
	defer func() {
		err := recover()
		if err != nil {
			log.Printf("[ERROR] %s", err)
			err = errors.New("bot crashed")
			weather = ""
		}
	}()

	if resp.StatusCode != 200 {
		err = fmt.Errorf("bot received error status code %d", resp.StatusCode)
		return "", err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		err = errors.New("bot received corrupt data")
		return "", err
	}

	var result map[string]interface{}
	json.Unmarshal([]byte(body), &result)

	weatherMap := result["weather"].([]interface{})
	weatherArr := weatherMap[0].(map[string]interface{})
	weather = weatherArr["description"].(string)
	weatherMain := result["main"].(map[string]interface{})
	weatherTemp := weatherMain["temp"].(float64)
	weather = fmt.Sprintf("Hi %s %s!\n\nWeather in %s is %s, with temperature of %0.2f degree Celsius.", firstName, lastName, formattedAddress, weather, weatherTemp)
	return weather, nil
}

func getLatLon(address string) (geocode Address, err error) {
	token := os.Getenv("MAPS_API_TOKEN")
	address = strings.ReplaceAll(address, " ", "%20")
	apiURL := "https://maps.googleapis.com/maps/api/geocode/json?address=" + address + "&key=" + token
	// log.Printf("[INFO] URL is: %s", apiURL)
	resp, err := http.Get(apiURL)
	// log.Printf("[INFO] Response is: %s", resp.Header)
	if err != nil {
		err = errors.New("location not found")
		return geocode, err
	}
	defer resp.Body.Close()
	defer func() {
		err := recover()
		if err != nil {
			// log.Printf("[ERROR] %s", err)
			err = errors.New("bot crashed")
			geocode.SetLatitude("")
			geocode.SetLongitude("")
			geocode.SetFormattedAddress("")
		}
	}()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		err = errors.New("bot received corrupt data")
		return geocode, err
	}

	var result map[string]interface{}
	json.Unmarshal([]byte(body), &result)

	results := result["results"].([]interface{})[0].(map[string]interface{})
	location := results["geometry"].(map[string]interface{})["location"].(map[string]interface{})
	formattedAddress := results["formatted_address"].(string)
	lat := fmt.Sprintf("%f", location["lat"].(float64))
	lon := fmt.Sprintf("%f", location["lng"].(float64))
	geocode.SetFormattedAddress(formattedAddress)
	geocode.SetLatitude(lat)
	geocode.SetLongitude(lon)
	return geocode, nil
}
