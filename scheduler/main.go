package main

import (
	"flag"
	"time"
	"fmt"
	"github.com/satori/go.uuid"
	"encoding/json"
	"github.com/golang/glog"
	"github.com/micro/go-micro/broker"
	//"github.com/micro/go-micro/cmd"
	"gitlab.com/greatercommons/youtube-drizzler/config"
	gpubsub "github.com/micro/go-plugins/broker/googlepubsub"

	"gitlab.com/greatercommons/youtube-drizzler/scheduler/storage"
	redislib "github.com/garyburd/redigo/redis"
	"gitlab.com/greatercommons/youtube-drizzler/scheduler/scheduler"

)



type TaskYoutube struct {
	TaskID 			string
	TaskItems 		[]storage.YouTubeVideo
	TaskFreq        int
}
var (
	consulPath = "kv/:youtube-drizzler/config"
	cfg youtubeDrizzlerConfig
	s scheduler.Scheduler
	b broker.Broker
)

type youtubeDrizzlerConfig struct {
	websiteTasksTopic string	`json:"websiteTasksTopic"`
	downloadWebsiteTasksTopic string	`json:"downloadWebsiteTasksTopic"`
	youtubeDrizzlerProjectID string	`json:"youtubeDrizzlerProjectID"`
}

func init()  {

	var c youtubeDrizzlerConfig
	cf := config.LoadJSONFromConsulKV(consulPath, c)

	val := cf.(map[string]interface {})

	cfg.websiteTasksTopic = val["websiteTasksTopic"].(string)

	cfg.downloadWebsiteTasksTopic = val["downloadWebsiteTasksTopic"].(string)

	cfg.youtubeDrizzlerProjectID = val["youtubeDrizzlerProjectID"].(string)

}

func pubDownLoadTasks(pub storage.YouTubeVideo) {
	data, _ := json.Marshal(pub)
	taskID := uuid.Must(uuid.NewV4()).String()
	msg := &broker.Message{
		Header: map[string]string{
			"id": fmt.Sprintf("%s", taskID),
		},
		Body: data,
	}
	if err := b.Publish(cfg.downloadWebsiteTasksTopic, msg); err != nil {

			glog.Infof("[scheduler pub] failed: %v %v", cfg.downloadWebsiteTasksTopic, err)
	} else {
			glog.Infof("[scheduler pub][%v] pubbed message:[%v] Header[%v]\n", cfg.downloadWebsiteTasksTopic, pub, taskID)
	}

}

// a subscription which receives all the messages
func sub() {
	task := TaskYoutube{}
	_, err := b.Subscribe(cfg.websiteTasksTopic, func(p broker.Publication) (error) {

		if err := json.Unmarshal(p.Message().Body, &task); err != nil {
			glog.Fatalf("Unable to parse JSON from sub '%s': %s", string(p.Message().Body), err)
		}
		glog.Infof("TaskID[%v]TaskFreq[%v]TaskItems[%v]\n", task.TaskID, task.TaskFreq, task.TaskItems)

		for _, v := range task.TaskItems {
		// Start a task with arguments
			if _, err := s.RunAfterYoutube(time.Duration(task.TaskFreq)*time.Second, pubDownLoadTasks, v, v); err != nil {
				glog.Fatal(err)
			}
		}

		return nil
	})
	if err != nil {
		glog.Error(err)
	}
	//pubDownLoadTasks(b, p.Message())
}

func walkQueue()  {
	tick := time.NewTicker(time.Second*60)
	i := 1
	for _ = range tick.C {
		glog.Info(i)
		i ++
	}
}

func main() {
	//cmd.Init()
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

	db := storage.NewRedis("",redislib.DialOption{}, false)

	s = scheduler.New(db)

/*	dob, _ := time.Parse(DateLayout, time.Now().Format(DateLayout))
	person := Person{
		ID:          "123-456",
		Name:        "John Smith 2",
		DateOfBirth: dob,
		Gender:      Male,
	}

	// Start a task with arguments
	if _, err := s.RunEvery(5*time.Second, CheckIfBirthday, person); err != nil {
		glog.Fatal(err)
	}*/
	sub()
	s.Start()
	s.Wait()

}

type Gender int

const DateLayout = "2006-01-02"
const (
	Male Gender = iota
	Female
)

type Person struct {
	ID          string
	Name        string
	DateOfBirth time.Time
	Gender      Gender
}

func CheckIfBirthday(person Person) {
	if time.Now().Format(DateLayout) == person.DateOfBirth.Format(DateLayout) {
		glog.Infof("Happy birthday,", person.Name)
		return
	}
	glog.Infof("Still waiting for your birthday")
}