package main

import (
	"fmt"
	"log"

	"github.com/micro/go-micro/broker"
	"github.com/micro/go-micro/cmd"

	gpubsub "github.com/micro/go-plugins/broker/googlepubsub"
)

var (
	//topic = "go.micro.topic.foo"
	topic = "tasks_youtube"
	topicdownload = "download_tasks_youtube"
	projectid = "myapisample-192900"
)

// Example of a shared subscription which receives a subset of messages
func sharedSub() {
	_, err := broker.Subscribe(topic, func(p broker.Publication) error {
		fmt.Println("[scheduler sub] received message:", string(p.Message().Body), "header", p.Message().Header)
		return nil
	}, broker.Queue("consumer"))
	if err != nil {
		fmt.Println(err)
	}
}

func pubDownLoadTasks(p broker.Broker, pub broker.Publication) {


		if err := p.Publish(topicdownload, pub.Message()); err != nil {
			log.Printf("[scheduler pub] failed: %v %v", topicdownload, err)
		} else {
			fmt.Printf("[scheduler pub][%v] pubbed message:[%v] Header[%v]\n", topicdownload, string(pub.Message().Body), pub.Message().Header)
		}

}

// Example of a subscription which receives all the messages
func sub(b broker.Broker) {
	_, err := b.Subscribe(topic, func(p broker.Publication) (error) {
		fmt.Printf("[sub][%v] received message:[%v] |header[%v]\n", topic, string(p.Message().Body),  p.Message().Header)
		pubDownLoadTasks(b, p)
		return nil
	})
	if err != nil {
		fmt.Println(err)
	}
	//pubDownLoadTasks(b, p.Message())
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
