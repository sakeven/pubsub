package main

import (
    "log"
    "runtime"
    "time"
)

var pubsub *Pubsub

func GetHello() {
    channel := pubsub.Register("Hello")
    //sub := channel.Subscribe()
    defer func() {
        log.Println("unregister")
        pubsub.Unregister("Hello")
    }()
    log.Println("hhhahs")
    //time.Sleep(1 * time.Second)
    channel.Publish("Hello")

    // time.Sleep(1 * time.Second)
    channel.Publish("World")

}

func GetHello2(name string) {
    channel := pubsub.Register("Hello")
    sub := channel.Subscribe()
    defer sub.Close()
    // sub.Close()

    time.Sleep(1 * time.Second)
    dataChan := sub.ReadChan()
    log.Println("recive", name, <-dataChan)

    runtime.Gosched()

    log.Println("recive", name, <-dataChan)
}

func main() {
    pubsub = New()
    go GetHello()

    // time.Sleep(3 * time.Second)
    go GetHello2("sake")
    // time.Sleep(1 * time.Second)

    go GetHello2("sakeven")

    for {
        runtime.Gosched()
    }
}
