package astronomia

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	// "log"
	"net/http"
	"os"
	"strings"
)

// Address is a struct to save the formatted address, latitude and longitudw.
type Address struct {
	FormattedAddress string
	Latitude         string
	Longitude        string
}

// SetFormattedAddress set formatted address.
func (a *Address) SetFormattedAddress(address string) {
	a.FormattedAddress = address
}

// SetLatitude sets latitude.
func (a *Address) SetLatitude(lat string) {
	a.Latitude = lat
}

// SetLongitude sets longitude.
func (a *Address) SetLongitude(lon string) {
	a.Longitude = lon
}

// GetWeather gets weather report.
func GetWeather(firstName, lastName, address string) (weatherMessage string, err error) {
	// OpenWeatherMap API token.
	token := os.Getenv("API_TOKEN")

	// Get latitude and longitude of address supplied.
	location, err := getLatLon(address)
	if err != nil {
		return fmt.Sprint(err), err
	}
	formattedAddress, lat, lon := location.FormattedAddress, location.Latitude, location.Longitude

	// Call OpenWeatherMap API.
	resp, err := http.Get("http://api.openweathermap.org/data/2.5/weather?lat=" + lat + "&lon=" + lon + "&appid=" + token + "&units=metric")
	if err != nil {
		err = errors.New("weather not found for this location") // Sorry!
		return "Network error!", err
	}

	// Always close the response body.
	defer resp.Body.Close()

	// Always recover from panic.
	defer func() {
		err := recover()
		if err != nil {
			err = errors.New(fmt.Sprintf("bot crashed with stacktrace %s", err))
			weatherMessage = "the bot crashed"
		}
	}()

	// If weather not found, then, ...
	if resp.StatusCode != 200 {
		// Spit out the response status.
		err = fmt.Errorf("the bot received error status code %d: %s", resp.StatusCode, resp.Status)
		return fmt.Sprint(err), err
	}

	// Parse response body, but if we can't, ...
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		// Say we received corrupt data LOL.
		err = errors.New("the bot received corrupt data")
		return fmt.Sprint(err), err
	}

	// Unmarshal the bytes.
	var result map[string]interface{}
	json.Unmarshal([]byte(body), &result)

	// Get weather condition.
	weatherResult := result["weather"].([]interface{})
	weather := weatherResult[0].(map[string]interface{})
	condition := weather["description"].(string)

	// Get temperature, feels like, pressure.
	weatherMainResult := result["main"].(map[string]interface{})
	temperature := weatherMainResult["temp"].(float64)
	feelsLike := weatherMainResult["feels_like"].(float64)
	humidity := weatherMainResult["humidity"].(float64)

	// Format weather message.
	weatherMessage = fmt.Sprintf(
		"Hi %s %s! Weather in %s is %s, with temperature of <b>%0.2f\u00B0C</b>. It feels like <b>%0.2f\u00B0C</b> with <b>%.f%%</b> humidity.",
		firstName, lastName, formattedAddress, condition, temperature, feelsLike, humidity,
	)
	return weatherMessage, nil
}

func getLatLon(address string) (geocode Address, err error) {
	// Maps API token.
	token := os.Getenv("MAPS_API_TOKEN")

	// Parse address.
	address = strings.ReplaceAll(address, " ", "%20")

	// Call Maps Geocoding API.
	url := "https://maps.googleapis.com/maps/api/geocode/json?address=" + address + "&key=" + token
	resp, err := http.Get(url)
	if err != nil {
		err = errors.New("location not found")
		return geocode, err
	}

	// Always close the response body.
	defer resp.Body.Close()

	// Always recover from panic.
	defer func() {
		err := recover()
		if err != nil {
			err = errors.New("the bot crashed")
			geocode.SetLatitude("")
			geocode.SetLongitude("")
			geocode.SetFormattedAddress("")
		}
	}()

	// Parse response body, but if we can't, ...
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		// Say we received corrupt data LOL.
		err = errors.New("the bot received corrupt data")
		return geocode, err
	}

	// Unmarshal the bytes.
	var result map[string]interface{}
	json.Unmarshal([]byte(body), &result)

	// Get formatted address, latitude and longitude.
	results := result["results"].([]interface{})[0].(map[string]interface{})
	location := results["geometry"].(map[string]interface{})["location"].(map[string]interface{})
	formattedAddress := results["formatted_address"].(string)
	lat := fmt.Sprintf("%f", location["lat"].(float64))
	lon := fmt.Sprintf("%f", location["lng"].(float64))

	// Set them in the struct.
	geocode.SetFormattedAddress(formattedAddress)
	geocode.SetLatitude(lat)
	geocode.SetLongitude(lon)
	return geocode, nil
}
