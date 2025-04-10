package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/spf13/viper"
)

type Story struct {
	Uid          string `json:"uid"`
	Anchor       string `json:"anchor"`
	CanonicalUrl string `json:"canonical_url"`
	Title        string `json:"title"`
	Credits      string `json:"credits"`
	Content      string `json:"content"`
	StoryDate    string `json:"story_date"`
	IsLanding    bool   `json:"is_landing"`
	UpdatedAt    string `json:"updated_at"`
	Url          string `json:"url"`
	Hash         string `json:"hash"`
}

type SiteData struct {
	Title       string  `json:"title"`
	Entity      string  `json:"entity"`
	Locale      string  `json:"locale"`
	SiteName    string  `json:"site_name"`
	LogoUrl     string  `json:"logo_url"`
	HomepageUrl string  `json:"homepage_url"`
	Stories     []Story `json:"stories"`
	ErrorPage   string  `json:"error_page"`
}

func fetch_local() ([]byte, error) {
	fileName := "local.json"
	jsonFile, err := os.Open(fileName)

	if err != nil {
		fmt.Println(err)
	}

	log.Printf("Fetching local response %v\n", fileName)

	defer jsonFile.Close()

	byteValue, err := io.ReadAll(jsonFile)

	if err != nil {
		return byteValue, errors.New("unable to read file data")
	}

	return byteValue, nil
}

func fetch_remote(url, token string) ([]byte, error) {

	apiClient := http.Client{
		Timeout: time.Second * 2, // Timeout after 2 seconds
	}

	log.Printf("Fetching data from %v\n", url)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.New("HTTP Error")
	}

	req.Header.Set("User-Agent", "NRTK SyncGo v0.1")
	req.Header.Set("Authorization", fmt.Sprintf("Token %v", token))

	response, getErr := apiClient.Do(req)
	if getErr != nil || response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request Error: %v", response.StatusCode)
	}

	if response.Body != nil {
		defer response.Body.Close()
	}

	body, readErr := io.ReadAll(response.Body)
	if readErr != nil {
		return nil, readErr
	}

	return body, nil
}

func main() {

	log.SetPrefix("nrtk-sync: ")
	log.SetFlags(0)

	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")

	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error config file: %w", err))
	}

	var api_response []byte
	var fetchError error

	if viper.GetBool("IS_REMOTE") {
		api_response, fetchError = fetch_remote(viper.GetString("NRTK_API_URL"), viper.GetString("NRTK_API_TOKEN"))
	} else {
		api_response, fetchError = fetch_local()
	}

	if fetchError != nil {
		log.Fatal(fetchError)

	} else {

		api_response_data := SiteData{}
		jsonErr := json.Unmarshal(api_response, &api_response_data)

		if jsonErr != nil {
			log.Fatal("unable to parse JSON data")
			log.Fatal(jsonErr)
		} else {
			log.Printf("Sync content for %v\n", api_response_data.SiteName)
		}

	}

}
