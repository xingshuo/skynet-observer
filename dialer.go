package main

import (
	"fmt"
	"log"
	"net"
	"time"
)

const (
	ReadBufferLen = 4096
)

type Dialer struct {
	observer  *Observer
	conn      net.Conn
	onMessage func([]byte)
	rbuf      [ReadBufferLen]byte
}

func (d *Dialer) Connect(address string, timeout time.Duration) {
	d.Close()
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		log.Fatalf("connect %s failed,%v\n", address, err)
	}
	d.conn = conn
	go d.loopRead()
}

func (d *Dialer) loopRead() {
	for {
		n, err := d.conn.Read(d.rbuf[:])
		if err != nil {
			d.observer.Quit(fmt.Sprintf("socket read err,%v", err))
			break
		}
		if n <= 0 {
			d.observer.Quit("socket read EOF")
			break
		}
		if d.onMessage != nil {
			d.onMessage(d.rbuf[:n])
		}
	}
}

func (d *Dialer) Send(b []byte) {
	_, err := d.conn.Write(b)
	if err != nil {
		d.observer.Quit(fmt.Sprintf("socket write err,%v", err))
	}
}

func (d *Dialer) Close() {
	if d.conn != nil {
		d.conn.Close()
	}
}
