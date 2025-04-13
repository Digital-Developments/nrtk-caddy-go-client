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
		defer f.Close()
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

func (s *Story) GetSitemapItem() string {
	priority := .8
	if s.Anchor == "index" {
		priority = 1
	}
	return fmt.Sprintf("<url><loc>%v</loc><lastmod>%v+00:00</lastmod><priority>%v</priority></url>", s.CanonicalUrl, s.UpdatedAt[:19], priority)
}

func (s Story) GetFilePath() string {
	return fmt.Sprintf("%v%v%v", viper.GetString("ContentDir"), s.Anchor, viper.GetString("ContentFileExtension"))
}

func (s Story) GetContent() []byte {
	return []byte(s.Content)
}

type ContentFile struct {
	FileName string
	Content  string
}

func (f ContentFile) GetFilePath() string {
	return fmt.Sprintf("%v%v", viper.GetString("ContentDir"), f.FileName)
}

func (f ContentFile) GetContent() []byte {
	return []byte(f.Content)
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

func (sd *SiteData) GetSitemap() string {

	sitemap := "<?xml version=\"1.0\" encoding=\"UTF-8\"?><urlset xmlns=\"http://www.sitemaps.org/schemas/sitemap/0.9\" xmlns:xsi=\"http://www.w3.org/2001/XMLSchema-instance\" xsi:schemaLocation=\"http://www.sitemaps.org/schemas/sitemap/0.9 http://www.sitemaps.org/schemas/sitemap/0.9/sitemap.xsd\"><!-- Created by Newsroom Toolkit www.newsroomtoolkit.com -->"

	for _, story := range sd.Stories {
		sitemap += story.GetSitemapItem()
	}

	return sitemap + "</urlset>"

}

type MetaObject struct {
	Title       string    `json:"title"`
	Entity      string    `json:"entity"`
	HomepageUrl string    `json:"homepage_url"`
	Stories     []Story   `json:"stories"`
	Checksum    string    `json:"checksum"`
	UpdatedAt   time.Time `json:"updated_at"`
	IsExpired   bool      `json:"-"`
}

func (m *MetaObject) IsUpdateNeeded() (bool, error) {

	local_json, err := readJSONFile(viper.GetString("MetaPath"))

	if err != nil {
		return true, err
	} else {
		current_meta := MetaObject{}
		jsonErr := json.Unmarshal(local_json, &current_meta)
		if jsonErr != nil {
			return true, jsonErr
		} else {
			current_meta.IsExpired = current_meta.Checksum != m.Checksum
			if current_meta.IsExpired {
				SaveFile(current_meta)
			}
			return current_meta.IsExpired, nil
		}
	}

}

func (m *MetaObject) SetChecksum(data []byte) {
	h := sha256.New()
	h.Write(data)
	m.Checksum = fmt.Sprintf("%x", h.Sum(nil))
	m.UpdatedAt = time.Now()
}

func (m MetaObject) GetFilePath() string {
	if m.IsExpired {
		return fmt.Sprintf("%vmeta.%v.json", viper.GetString("SnapshotDir"), m.Checksum)
	}
	return viper.GetString("MetaPath")
}

func (m MetaObject) GetContent() []byte {
	json_dump, _ := json.Marshal(m)
	return json_dump
}

func readJSONFile(filePath string) ([]byte, error) {
	jsonFile, err := os.Open(filePath)

	log.Printf("Reading JSON from %v\n", filePath)

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

	bin_dir_err := os.MkdirAll(viper.GetString("SnapshotDir"), 0755)
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

	meta_object := MetaObject{
		Title:       sync_data.Title,
		Entity:      sync_data.Entity,
		HomepageUrl: sync_data.HomepageUrl,
		Stories:     sync_data.Stories,
	}

	meta_object.SetChecksum(api_response)

	result, _ := meta_object.IsUpdateNeeded()

	if result || viper.GetBool("IS_FORCE_UPDATE") {

		log.Printf("Sync content for %v with %v stories (IS_FORCE_UPDATE=%v)\n", sync_data.SiteName, len(sync_data.Stories), viper.GetBool("IS_FORCE_UPDATE"))

		SaveFile(meta_object)

		for _, story := range sync_data.Stories {
			e := SaveFile(story)
			if e != nil {
				fmt.Println(e)
			}
		}

		error_page := ContentFile{
			FileName: "error" + viper.GetString("ContentFileExtension"),
			Content:  sync_data.ErrorPage,
		}
		SaveFile(error_page)

		sitemap := ContentFile{
			FileName: "sitemap.xml",
			Content:  sync_data.GetSitemap(),
		}
		SaveFile(sitemap)

	} else {
		log.Printf("Nothing to update")
	}

}

func run() {
	var api_response []byte
	var fetchError error

	if viper.GetBool("IS_REMOTE") {
		api_response, fetchError = fetch_remote(viper.GetString("NRTK_API_URL"), viper.GetString("NRTK_API_TOKEN"))
	} else {
		api_response, fetchError = readJSONFile("local.json")
	}

	if fetchError != nil {
		panic(fetchError)
	} else {
		sync(api_response)
	}
}

func main() {

	log.SetPrefix("nrtk-sync: ")
	log.SetFlags(0)

	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")

	viper.SetDefault("AppDir", ".nrtk/")
	viper.SetDefault("ContentDir", viper.GetString("AppDir")+"www/")
	viper.SetDefault("SnapshotDir", viper.GetString("AppDir")+"snapshot/")
	viper.SetDefault("MetaPath", viper.GetString("AppDir")+"meta.json")
	viper.SetDefault("ContentFileExtension", ".html")

	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error config file: %w", err))
	}

	if viper.GetInt64("INFINITY") > 0 {
		for {
			sleep_timer := time.Duration(viper.GetInt("INFINITY")) * time.Millisecond
			run()
			log.Printf("Sleeping for %v seconds", sleep_timer.Seconds())
			time.Sleep(sleep_timer)
		}
	} else {
		run()
	}

}
