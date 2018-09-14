package main

import (
	"fmt"
	"log"
	"time"

	"github.com/everestmz/maxfuzz/internal/docker"
)

func main() {
	err := docker.Init()
	if err != nil {
		panic(err)
	}
	stopChan := make(chan bool)
	go func() {
		log.Println("Goroutine")
		time.Sleep(time.Second * time.Duration(3))
		log.Println("Done sleep")
		stopChan <- true
		log.Println("Done")
	}()
	clusterConfig, err := docker.CreateFuzzer("vulnerable", stopChan)
	if err != nil {
		panic(err)
	}
	log.Println(fmt.Sprintf("%+v", clusterConfig))
}
