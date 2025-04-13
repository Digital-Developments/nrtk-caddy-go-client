package main

import (
	"crypto/sha256"
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

type LocalWriter interface {
	GetFilePath() string
	GetContent() []byte
}

func SaveFile(any interface{}) error {

	if obj, ok := any.(LocalWriter); ok {
		f, err := os.OpenFile(obj.GetFilePath(), os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			return fmt.Errorf("unable to open %v for writing\n", obj.GetFilePath())
		}

		log.Printf("Saving %v\n", obj.GetFilePath())
		_, e := f.Write(obj.GetContent())
		if e != nil {
			return e
		}
		f.Close()
		return nil
	}

	return fmt.Errorf("LocalWriter interface is not supported")

}

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

func (s Story) GetFilePath() string {
	return fmt.Sprintf("%v%v%v", viper.GetString("ContentDir"), s.Anchor, viper.GetString("ContentFileExtension"))
}

func (s Story) GetContent() []byte {
	return []byte(s.Content)
}

type ErrorPage struct {
	Content string
}

func (e ErrorPage) GetFilePath() string {
	return fmt.Sprintf("%v%v%v", viper.GetString("ContentDir"), "error", viper.GetString("ContentFileExtension"))
}

func (e ErrorPage) GetContent() []byte {
	return []byte(e.Content)
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

type MetaObject struct {
	Title       string    `json:"title"`
	Entity      string    `json:"entity"`
	HomepageUrl string    `json:"homepage_url"`
	Stories     []Story   `json:"stories"`
	Checksum    string    `json:"checksum"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (m *MetaObject) SetChecksum(data []byte) {
	h := sha256.New()
	h.Write(data)
	m.Checksum = fmt.Sprintf("%x", h.Sum(nil))
	log.Printf("Content Checksum: %v\n", m.Checksum)

	m.UpdatedAt = time.Now()
	log.Printf("Meta Updated: %v\n", m.UpdatedAt)
}

func (m MetaObject) GetFilePath() string {
	return viper.GetString("MetaPath")
}

func (m MetaObject) GetContent() []byte {
	json_dump, _ := json.Marshal(m)
	return json_dump
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

func create_dirs() error {

	content_dir_err := os.MkdirAll(viper.GetString("ContentDir"), 0755)
	if content_dir_err != nil {
		return content_dir_err
	}

	bin_dir_err := os.MkdirAll(viper.GetString("BinDir"), 0755)
	if bin_dir_err != nil {
		return bin_dir_err
	}

	return nil

}

func sync(api_response []byte) {

	sync_data := SiteData{}
	jsonErr := json.Unmarshal(api_response, &sync_data)

	if jsonErr != nil {
		log.Fatal(jsonErr)
		panic("unable to parse JSON data")
	}

	err := create_dirs()
	if err != nil {
		log.Fatal(err)
		panic("unable to create app dirs")
	}

	log.Printf("Sync content for %v with %v stories\n", sync_data.SiteName, len(sync_data.Stories))

	error_page := ErrorPage{
		Content: sync_data.ErrorPage,
	}
	SaveFile(error_page)

	meta_object := MetaObject{
		Title:       sync_data.Title,
		Entity:      sync_data.Entity,
		HomepageUrl: sync_data.HomepageUrl,
	}

	meta_object.SetChecksum(api_response)

	for _, story := range sync_data.Stories {
		e := SaveFile(story)
		if e != nil {
			fmt.Println(e)
		}
	}

	SaveFile(meta_object)

}

func main() {

	log.SetPrefix("nrtk-sync: ")
	log.SetFlags(0)

	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")

	viper.SetDefault("AppDir", ".nrtk/")
	viper.SetDefault("ContentDir", viper.GetString("AppDir")+"www/")
	viper.SetDefault("BinDir", viper.GetString("AppDir")+"bin/")
	viper.SetDefault("MetaPath", viper.GetString("AppDir")+"meta.json")
	viper.SetDefault("ContentFileExtension", ".html")

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
		panic(fetchError)
	} else {
		sync(api_response)
	}

}
