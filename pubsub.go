package main

import (
    "log"
    "sync"
    "time"
)

type Pubsub struct {
    client map[string]*Channel
    lock   sync.Mutex
}

func New() *Pubsub {
    return &Pubsub{
        client: make(map[string]*Channel),
    }
}

func (p *Pubsub) Register(topic string) *Channel {

    p.lock.Lock()
    defer p.lock.Unlock()

    if c, ok := p.client[topic]; ok {
        return c
    }

    c := NewChannel(p, topic)
    go c.start()
    p.client[topic] = c
    return c
}

func (p *Pubsub) Unregister(topic string) {
    p.lock.Lock()
    defer p.lock.Unlock()

    c, ok := p.client[topic]
    if !ok {
        return
    }

    c.close()
    delete(p.client, topic)
}

func (p *Pubsub) Lookup(topic string) *Channel {
    p.lock.Lock()
    defer p.lock.Unlock()

    if c, ok := p.client[topic]; ok {
        return c
    }

    return nil
}

type Channel struct {
    pubsub        *Pubsub
    topic         string
    closed        chan bool
    history       []interface{}
    unsubscribe   chan *Subscription
    subscribe     chan *Subscription
    subscriptions map[*Subscription]bool
    broadcast     chan interface{}
}

func NewChannel(pubsub *Pubsub, topic string) *Channel {
    return &Channel{
        pubsub:        pubsub,
        topic:         topic,
        unsubscribe:   make(chan *Subscription),
        subscribe:     make(chan *Subscription),
        subscriptions: make(map[*Subscription]bool),
        broadcast:     make(chan interface{}),
    }
}

func (c *Channel) Publish(data interface{}) {
    go func() { c.broadcast <- data }()
}

func (c *Channel) close() {
    go func() { c.closed <- true }()
}

func (c *Channel) Subscribe() *Subscription {
    sub := NewSubscription(c)
    c.subscribe <- sub
    return sub
}
func (c *Channel) Unsubscribe(sub *Subscription) {
    go func() { c.unsubscribe <- sub }()
}

func (c *Channel) start() {
    defer func() {
        if e := recover(); e != nil {
            log.Printf("Channel %s panic %v\n", c.topic, e)
        }
    }()

    var timeout <-chan time.Time
    // if c.timeout > 0 {
    timeout = time.After(60 * time.Minute)
    // }

    for {
        select {
        case sub := <-c.unsubscribe:
            delete(c.subscriptions, sub)
        case sub := <-c.subscribe:
            c.subscriptions[sub] = true

            if len(c.history) > 0 {
                history := make([]interface{}, len(c.history))
                copy(history, c.history)
                c.replay(sub, history)
            }

        case msg := <-c.broadcast:
            for sub := range c.subscriptions {
                if sub.Closed() {
                    continue
                }

                select {
                case sub.send <- msg:
                }
            }

            c.history = append(c.history, msg)
        case <-c.closed:
            for sub, _ := range c.subscriptions {
                sub.Close()
            }

        case <-timeout:
            pubsub.Unregister(c.topic)
        }
    }
}

func (c *Channel) replay(sub *Subscription, history []interface{}) {
    go func() {
        for _, msg := range history {
            sub.send <- msg
        }
    }()

}

type Subscription struct {
    channel     *Channel
    closed      bool
    closeNotify chan bool
    send        chan interface{}
}

func NewSubscription(c *Channel) *Subscription {
    return &Subscription{
        channel: c,
        send:    make(chan interface{}),
    }
}

func (s *Subscription) Closed() bool {
    return s.closed
}
func (s *Subscription) Close() {
    if s.closed {
        return
    }

    s.closed = true
    close(s.send)
    s.channel.Unsubscribe(s)
    go func() { s.closeNotify <- true }()
}

func (s *Subscription) ReadChan() <-chan interface{} {
    return s.send
}

func (s *Subscription) CloseNotify() <-chan bool {
    return s.closeNotify
}

// func (s *Subscription) Read() interface{} {
//     return <-s.send
// }
