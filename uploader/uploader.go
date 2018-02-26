package main

import (
	"encoding/gob"
	"errors"
	"flag"
	"fmt"
	"hash/fnv"

	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/youtube/v3"
	"github.com/micro-youtube-drizzler/config"
	"gitlab.com/greatercommons/youtube-drizzler/scheduler/storage"
	"io/ioutil"
	"github.com/golang/glog"
	redislib "github.com/garyburd/redigo/redis"
)
var (
	cacheToken = flag.Bool("cachetoken", true, "cache the OAuth 2.0 token")
	filename = "./videofiles/my_cute_boy.mp4"
	consulPath = "kv/:youtube-drizzler/config"
	cfg youtubeDrizzlerConfig
)

type youtubeDrizzlerConfig struct {
	websiteTasksTopic 			string	`json:"websiteTasksTopic"`
	downloadWebsiteTasksTopic 	string	`json:"downloadWebsiteTasksTopic"`
	youtubeDrizzlerProjectID 	string	`json:"youtubeDrizzlerProjectID"`
	videoFilesPath				string	`json:"videoFilesPath"`
	videoFilesLink				string	`json:"videoFilesLink"`
	clientID					string	`json:"clientID"`
	clientSecret				string	`json:"clientSecret"`
}

func init()  {

	var c youtubeDrizzlerConfig
	cf := config.LoadJSONFromConsulKV(consulPath, c)
	val := cf.(map[string]interface {})

	cfg.youtubeDrizzlerProjectID = val["youtubeDrizzlerProjectID"].(string)
	cfg.videoFilesPath = val["videoFilesPath"].(string)
	cfg.videoFilesLink = val["videoFilesLink"].(string)
	cfg.clientID = val["clientID"].(string)
	cfg.clientSecret = val["clientSecret"].(string)

}

func main() {
	flag.Parse()
	config := &oauth2.Config{
		ClientID:     cfg.clientID,
		ClientSecret: cfg.clientSecret,
		Endpoint:     google.Endpoint,
		Scopes:       []string{youtube.YoutubeUploadScope},
	}

	ctx := context.Background()

	client := newOAuthClient(ctx, config)

	service, err := youtube.New(client)
	if err != nil {
		glog.Fatalf("Unable to create YouTube service: %v", err)
	}
	db := storage.NewRedis("",redislib.DialOption{}, false)
	tick := time.NewTicker(time.Second*60)

	for _ = range tick.C {
		videoFiles, err := ListDir(cfg.videoFilesLink, ".mp4")
		if err != nil {

		}
		for _, videoFile := range videoFiles {

			glog.Infof("process videoFile[%v]\n", videoFile)
			fileName := filepath.Base(videoFile)
			statusJob, err := db.GetJobStatus(fileName)
			if err != nil {
				glog.Fatalf("Error GetJobStatus Redis %v: %v\n", fileName, err)
			}
			if statusJob == "U" {
				glog.Errorf("Error Duplicate video ID %v\n", fileName)
				continue
			}

			videoFileMeta, err := db.GetVideoMeta(fileName)
			glog.Infof("[%v][%v]\n", fileName, videoFileMeta)

			if err != nil {
				glog.Errorf("Error Get Redis %v: %v\n", fileName, err)
			}
			file, err := os.Open(videoFile)
			if err != nil {
				glog.Fatalf("Error opening %v: %v", videoFile, err)
			}

			upload := &youtube.Video{
				Snippet: &youtube.VideoSnippet{
					Title:       videoFileMeta.Title,
					Description: videoFileMeta.Description, // can not use non-alpha-numeric characters
					CategoryId:  videoFileMeta.CategoryId,
				},

				Status: &youtube.VideoStatus{
					PrivacyStatus:       "public",
					PublicStatsViewable: true,
				},
			}

			// The API returns a 400 Bad Request response if tags is an empty string.
			upload.Snippet.Tags = []string{"test", "upload", "api"}

			call := service.Videos.Insert("snippet,status", upload)

			response, err := call.Media(file).Do()
			if err != nil {
				glog.Fatalf("Error making YouTube API call: %v", err)
			}
			fmt.Printf("Upload successful! Video ID: %v\n", response.Id)
			file.Close()
			err = db.SetJobStatus(fileName, "U")
			if err != nil {
				glog.Fatalf("Error set job status for [%v][%v]", fileName, err)
			}
			del := os.Remove(videoFile)
			if del != nil {
				glog.Errorf("Error del link file : %v", err)
			}
		}
	}
}

func ListDir(dirPth string, suffix string) (files []string, err error) {
	files = make([]string, 0, 10)

	dir, err := ioutil.ReadDir(dirPth)
	if err != nil {
		return nil, err
	}

	PthSep := string(os.PathSeparator)
	suffix = strings.ToUpper(suffix)

	for _, fi := range dir {
		if fi.IsDir() {
			continue
		}
		if strings.HasSuffix(strings.ToUpper(fi.Name()), suffix) {
			files = append(files, dirPth+PthSep+fi.Name())
		}
	}

	return files, nil
}
func newOAuthClient(ctx context.Context, config *oauth2.Config) *http.Client {
	cacheFile := tokenCacheFile(config)
	token, err := tokenFromFile(cacheFile)
	if err != nil {
		token = tokenFromWeb(ctx, config)
		saveToken(cacheFile, token)
	} else {
		glog.Infof("Using cached token %#v from %q", token, cacheFile)
	}

	return config.Client(ctx, token)
}

func tokenFromWeb(ctx context.Context, config *oauth2.Config) *oauth2.Token {
	ch := make(chan string)
	randState := fmt.Sprintf("st%d", time.Now().UnixNano())
	ts := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/favicon.ico" {
			http.Error(rw, "", 404)
			return
		}
		if req.FormValue("state") != randState {
			glog.Errorf("State doesn't match: req = %#v", req)
			http.Error(rw, "", 500)
			return
		}
		if code := req.FormValue("code"); code != "" {
			fmt.Fprintf(rw, "<h1>Success</h1>Authorized.")
			rw.(http.Flusher).Flush()
			ch <- code
			return
		}

		http.Error(rw, "", 500)
	}))
	defer ts.Close()

	config.RedirectURL = ts.URL
	authURL := config.AuthCodeURL(randState)
	go openURL(authURL)
	glog.Infof("Authorize this app at: %s", authURL)
	code := <-ch
	glog.Infof("Got code: %s", code)

	token, err := config.Exchange(ctx, code)
	if err != nil {
		glog.Fatalf("Token exchange error: %v", err)
	}
	return token
}

func openURL(url string) {
	try := []string{"xdg-open", "google-chrome", "open"}
	for _, bin := range try {
		err := exec.Command(bin, url).Run()
		if err == nil {
			return
		}
	}
	glog.Error("Error opening URL in browser.")
}
func tokenCacheFile(config *oauth2.Config) string {
	hash := fnv.New32a()
	hash.Write([]byte(config.ClientID))
	hash.Write([]byte(config.ClientSecret))
	hash.Write([]byte(strings.Join(config.Scopes, " ")))
	fn := fmt.Sprintf("go-api-demo-tok%v", hash.Sum32())
	return filepath.Join(osUserCacheDir(), url.QueryEscape(fn))
}
func osUserCacheDir() string {
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(os.Getenv("HOME"), "Library", "Caches")
	case "linux", "freebsd":
		return filepath.Join(os.Getenv("HOME"), ".cache")
	}

	return "."
}
func tokenFromFile(file string) (*oauth2.Token, error) {
	if !*cacheToken {
		return nil, errors.New("--cachetoken is false")
	}
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	t := new(oauth2.Token)
	err = gob.NewDecoder(f).Decode(t)
	return t, err
}

func saveToken(file string, token *oauth2.Token) {
	f, err := os.Create(file)
	if err != nil {
		glog.Errorf("Warning: failed to cache oauth token: %v", err)
		return
	}
	defer f.Close()
	gob.NewEncoder(f).Encode(token)
}