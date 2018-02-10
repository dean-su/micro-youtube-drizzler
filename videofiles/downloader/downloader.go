package main

import (
	"fmt"
	"log"

	"github.com/micro/go-micro/broker"
	"github.com/micro/go-micro/cmd"

	gpubsub "github.com/micro/go-plugins/broker/googlepubsub"
	"cloud.google.com/go/storage"
	"golang.org/x/net/context"
	//"google.golang.org/api/iterator"

	"io/ioutil"
)

var (
	//topic = "go.micro.topic.foo"
	topic = "tasks_youtube"
	topicdownload = "download_tasks_youtube"
	projectid = "myapisample-192900"

)

func download(pmes broker.Publication) error  {
	ctx := context.Background()
	// Creates a client.
	client, err := storage.NewClient(ctx)

	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	bucketid := pmes.Message().Header["buckets"]
	videoname := string(pmes.Message().Body)

	fmt.Println(bucketid, videoname)
	bh := client.Bucket(bucketid)
	// Next check if the bucket exists
	if _, err := bh.Attrs(ctx); err != nil {
		return err
	}
	obj := bh.Object(videoname)
	rc, err := obj.NewReader(ctx)
	if err != nil {
		err = fmt.Errorf("readFile: unable to open file from bucket %q, file %q: %v", bucketid, videoname, err)
		return err
	}

	defer rc.Close()
	slurp, err := ioutil.ReadAll(rc)
	if err != nil {
		fmt.Errorf("readFile: unable to read data from bucket %q, file %q: %v", bucketid, videoname, err)
		return err
	}
	outfile := "./videofiles/" + videoname

	err = ioutil.WriteFile(outfile, slurp, 0666)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	return nil
}

// Example of a subscription which receives all the messages
func sub(b broker.Broker) {
	_, err := b.Subscribe(topicdownload, func(p broker.Publication) error {
		fmt.Printf("[downloader sub][%v] received message:[%v] |header[%v]\n", topic, string(p.Message().Body),  p.Message().Header)
		download(p)
		return nil
	})
	if err != nil {
		fmt.Println(err)
	}
}

func main() {
	cmd.Init()
	opt := gpubsub.ProjectID(projectid)
	nbroker := gpubsub.NewBroker(opt)
	if err := nbroker.Init(); err != nil {
		log.Fatalf("Broker Init error: %v", err)
	}
	if err := nbroker.Connect(); err != nil {
		log.Fatalf("Broker Connect error: %v", err)
	}

	sub(nbroker)
	select {}
}
