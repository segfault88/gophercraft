package main

import (
	"encoding/binary"
	"fmt"
	"github.com/segfault88/gophercraft/graphics"
	"time"
)

var (
	varintBuff [binary.MaxVarintLen64]byte
	host       = "localhost"
	port       = 25565

	renderer *graphics.Renderer
	client   *Client
)

func main() {
	fmt.Println("Gophercraft!\n")

	var err error
	renderer, err = graphics.Init()

	if err != nil {
		panic("Couldn't initialize graphics! " + err.Error())
	}

	defer renderer.Shutdown()

	// ping the minecraft server to see if it is there before moving further
	json, err := Ping(host, port)
	fmt.Printf("Ping Response:\n%s\n", json)

	client, err = JoinServer(host, port)
	if err != nil {
		panic(err)
	}
	defer client.Shutdown()

	run()
}

func run() {
	// start the tick goroutine
	tick := make(chan bool)
	go tick_run(20, tick)

	for !renderer.ShouldClose() {
		select {
		case <-tick:
			renderer.DrawFrame()
		case packet := <-client.packets:
			handlePacket(packet)
		}
	}

}

func tick_run(everyMS int, tick chan bool) {
	sleepTime := time.Duration(everyMS) * time.Millisecond

	for {
		time.Sleep(sleepTime)
		tick <- true
	}
}

func handlePacket(packet *Packet) {
	switch packet.Id {
	case 0x0:
		keepalive, err := ParseKeepalive(packet)

		if err != nil {
			panic(err)
		}

		fmt.Printf("Keepalive was: %d\n", keepalive)

		// send it right back
		client.SendKeepAlive(keepalive)
	case 0x01:
		ParseJoinGame(packet)
	case 0x03:
		ParseTimeUpdate(packet)
	case 0x08:
		ParsePlayerPositionAndLook(packet)
	case 0x09:
		ParseItemHeldChange(packet)
	case 0x26:
		err := ParseMapChunkBulk(packet)

		if err != nil {
			panic(err)
		}
	case 0x39:
		ParsePlayerAbilities(packet)
	default:
		fmt.Printf("Packet: 0x%0x\tsize: %d\tnot handled\n", packet.Id, packet.Size)
	}
}

func checkError(err error) {
	if err != nil {
		panic(err.Error())
	}
}
