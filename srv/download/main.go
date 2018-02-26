package main

import (
	"fmt"
	"flag"
	proto "github.com/micro-youtube-drizzler/srv/download/proto"
	"github.com/micro/go-micro"
	"encoding/json"
	"github.com/golang/glog"
	"github.com/micro-youtube-drizzler/config"
	gpubsub "github.com/micro/go-plugins/broker/googlepubsub"
	"cloud.google.com/go/storage"
	"golang.org/x/net/context"
	"github.com/micro/go-micro/broker"
	"io/ioutil"
	"path/filepath"
	"os"
)

/*

Example usage of top level service initialisation

*/
type youtubeDrizzlerConfig struct {
	websiteTasksTopic 			string	`json:"websiteTasksTopic"`
	downloadWebsiteTasksTopic 	string	`json:"downloadWebsiteTasksTopic"`
	youtubeDrizzlerProjectID 	string	`json:"youtubeDrizzlerProjectID"`
	videoFilesPath				string	`json:"videoFilesPath"`
	videoFilesLink				string	`json:"videoFilesLink"`
}

type YouTubeVideo struct{
	GCloudBucketName		string
	GCloudFileName			string
	Title					string
	CategoryId				string
	Description 			string
	CourseURL	 			string
}


var (
	consulPath = "kv/:youtube-drizzler/config"
	cfg youtubeDrizzlerConfig
	b broker.Broker
)

func init()  {

	var c youtubeDrizzlerConfig
	cf := config.LoadJSONFromConsulKV(consulPath, c)

	val := cf.(map[string]interface {})

	cfg.websiteTasksTopic = val["websiteTasksTopic"].(string)

	cfg.downloadWebsiteTasksTopic = val["downloadWebsiteTasksTopic"].(string)

	cfg.youtubeDrizzlerProjectID = val["youtubeDrizzlerProjectID"].(string)
	cfg.videoFilesPath = val["videoFilesPath"].(string)

	cfg.videoFilesLink = val["videoFilesLink"].(string)

}

func DownloadVideo(pmes *YouTubeVideo){
	ctx := context.Background()
	// Creates a client.
	client, err := storage.NewClient(ctx)

	if err != nil {
		glog.Fatalf("Failed to create client: %v", err)
	}

	bucketID := pmes.GCloudBucketName
	videoName := pmes.GCloudFileName

	fmt.Println(bucketID, videoName)
	bh := client.Bucket(bucketID)
	// Next check if the bucket exists
	if _, err := bh.Attrs(ctx); err != nil {
		glog.Error(err)
	}
	obj := bh.Object(videoName)
	rc, err := obj.NewReader(ctx)
	if err != nil {
		glog.Errorf("readFile: unable to open file from bucket %q, file %q: %v", bucketID, videoName, err)
		return
	}

	defer rc.Close()
	slurp, err := ioutil.ReadAll(rc)
	if err != nil {
		glog.Errorf("readFile: unable to read data from bucket %q, file %q: %v", bucketID, videoName, err)
		return
	}
	glog.Info(cfg.videoFilesPath)
	_, err = config.PathExists(cfg.videoFilesPath)
	if err != nil {
		glog.Fatalf("Failed to create video path: %v\n", err)
	}
	_, err = config.PathExists(cfg.videoFilesLink)
	if err != nil {
		glog.Fatalf("Failed to create video link path: %v", err)
	}

	outfile := filepath.Join(cfg.videoFilesPath, videoName)
	linkOutfile := filepath.Join(cfg.videoFilesLink, videoName)

	err = ioutil.WriteFile(outfile, slurp, 0666)
	if err != nil {
		glog.Fatalf("Failed to create video file: %v", err)
	}

	err = os.Symlink(outfile, linkOutfile)
	/*if err != nil {
		glog.Fatalf("Failed to create video link file: %v", err)
	}*/
	glog.Infof("Outfile:", outfile)
	glog.Infof("Outfilelink:", linkOutfile)

}

type Download struct{}

func (d *Download) SetDownloadTask(ctx context.Context, req *proto.Request, rsp *proto.Result) error {
	rsp.AckResult = "Ack task: " + req.GCloudFileName
	videoFile := new(YouTubeVideo)
	videoFile.GCloudFileName = req.GCloudFileName
	videoFile.GCloudBucketName = req.GCloudBucketName
	DownloadVideo(videoFile)
	return nil
}

func sub() {
	videoFile := new(YouTubeVideo)
	_, err := b.Subscribe(cfg.downloadWebsiteTasksTopic, func(p broker.Publication) error {
		if err := json.Unmarshal(p.Message().Body, &videoFile); err != nil {
			glog.Fatalf("Unable to parse JSON from sub '%s': %s", string(p.Message().Body), err)
		}
		glog.Infof("[downloader sub][topic:%v][header:%v] received message:[%v]\n", cfg.downloadWebsiteTasksTopic, p.Message().Header, videoFile)

		DownloadVideo(videoFile)
		return nil
	})
	if err != nil {
		glog.Error(err)
	}
}

func main() {
	flag.Parse()
	defer glog.Flush()
	opt := gpubsub.ProjectID(cfg.youtubeDrizzlerProjectID)
	b = gpubsub.NewBroker(opt)
	if err := b.Init(); err != nil {
		glog.Fatalf("Broker Init error: %v", err)
	}
	if err := b.Connect(); err != nil {
		glog.Fatalf("Broker Connect error: %v", err)
	}
	sub()
	// Create a new service. Optionally include some options here.
	service := micro.NewService(
		micro.Name("download"),
		micro.Version("latest"),
		micro.Metadata(map[string]string{
			"type": "hello downloader",
		}),
	)

	// Init will parse the command line flags. Any flags set will
	// override the above settings. Options defined here will
	// override anything set on the command line.
	//service.Init()

	// By default we'll run the server unless the flags catch us

	// Setup the server

	// Register handler
	proto.RegisterDownloadHandler(service.Server(), new(Download))
	glog.Info("Download Service is running")
	// Run the server
	if err := service.Run(); err != nil {
		fmt.Println(err)
	}

}
