package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
)

import (
	// "github.com/go-gl/gl"
	glfw "github.com/go-gl/glfw3"
	// "github.com/go-gl/glu"
)

type Client struct {
	Host   string
	Port   int
	conn   *net.Conn
	reader *bufio.Reader
}

type Packet struct {
	Id   int
	Size int
	Data *bytes.Buffer
}

func Ping(host string, port int) (string, error) {
	client := Client{Host: host, Port: port}
	err := client.connect()

	fmt.Printf("Going to ping server %s:%d\n", host, port)

	if err != nil {
		return "", err
	}

	defer client.disconnect()

	client.SendHandshake(1)
	client.SendStatusRequest()

	packet := readPacket(client.reader)

	return parseStatusRequest(packet)
}

func (client *Client) connect() error {
	if client.Host == "" || client.Port == 0 {
		return errors.New("Host or client is blank! Please make sure to set these before trying to connect!")
	}

	conn, err := net.Dial("tcp", client.Host+":"+strconv.Itoa(client.Port))
	checkError(err)

	client.conn = &conn
	client.reader = bufio.NewReader(conn)

	return nil
}

func (client *Client) disconnect() {
	client.reader = nil
	(*client.conn).Close()
}

func joinServer(host string, port int, window *glfw.Window) {
	_ = host
	_ = port
	_ = window
	// conn := connect(host, port)
	// defer conn.Close()

	// fmt.Println("Going to join server")

	// sendHandshake(conn, host, port, 2)
	// sendLoginStart(conn)

	// tick := make(chan bool)

	// go func() {
	// 	for {
	// 		time.Sleep(40 * time.Millisecond)
	// 		tick <- true
	// 	}
	// }()

	// reader := bufio.NewReader(conn)
	// readLoginSuccess(reader)

	// fromServer := make(chan Packet)

	// go func() {
	// 	reader := bufio.NewReader(conn)

	// 	for {
	// 		size, err := binary.ReadUvarint(reader)
	// 		if err != nil {
	// 			fmt.Println("Error reading from connection, returning from reciver. Error: " + err.Error())
	// 			return
	// 		}

	// 		// ignore the packet type in the packet length since we read it too
	// 		size -= 1

	// 		packetType, err := binary.ReadUvarint(reader)
	// 		checkError(err)

	// 		data := make([]byte, size)
	// 		n, _ := reader.Read(data)

	// 		for n < int(size) {
	// 			// didn't read all the data, keep trying to read some more
	// 			nx, _ := reader.Read(data[n:])
	// 			n += nx
	// 		}

	// 		var packet Packet
	// 		packet.id = int(packetType)
	// 		packet.size = int(size)
	// 		packet.data = data

	// 		fromServer <- packet
	// 	}
	// }()

	// tickCount := 0

	// for {
	// 	select {
	// 	case <-tick:
	// 		tickCount += 1
	// 		fmt.Printf("Tick %d\n", tickCount)

	// 		frame(window)

	// 		if tickCount >= 200 {
	// 			return
	// 		}
	// 	case packet := <-fromServer:
	// 		fmt.Printf("Packet: 0x%0x\t\tsize: %d\n", packet.id, packet.size)

	// 		switch packet.id {
	// 		case 0:
	// 			fmt.Printf("Keep alive: %d\n", packet.data)

	// 			databuffer := bytes.NewBuffer(packet.data)
	// 			var keepalive int
	// 			binary.Read(databuffer, binary.BigEndian, &keepalive)

	// 			fmt.Printf("Keepalive was: %d\n", keepalive)

	// 			// send it right back
	// 			buf := new(bytes.Buffer)

	// 			writeVarint(buf, 0)         // packet id
	// 			writeData(buf, packet.data) // keep alive id

	// 			sendPacket(conn, buf)
	// 		}
	// 	}
	// }
}

func (client *Client) SendHandshake(state int) {
	buf := new(bytes.Buffer)

	writeVarint(buf, 0)                 // packet id
	writeVarint(buf, 4)                 // protocol version
	writeString(buf, client.Host)       // server address
	writeData(buf, uint16(client.Port)) // server port
	writeVarint(buf, int64(state))      // next state 1 for status, 2 for login

	sendPacket(*client.conn, buf)
}

func (client *Client) SendStatusRequest() {
	buf := new(bytes.Buffer)

	writeVarint(buf, 0) // packet id

	sendPacket(*client.conn, buf)
}

func sendLoginStart(conn net.Conn) {
	buf := new(bytes.Buffer)

	writeVarint(buf, 0)             // packet id
	writeString(buf, "gophercraft") // username

	sendPacket(conn, buf)
}

func parseStatusRequest(packet *Packet) (string, error) {
	return readString(packet.Data), nil
}

func parseLoginSuccess(packet *Packet) (username string, uuid string, err error) {
	if packet.Id != 2 {
		return "", "", errors.New("Expected packet id 2 (Login Success), but got: " + strconv.Itoa(int(packet.Id)))
	}

	fmt.Println("Got login success packet!")

	uuid = readString(packet.Data)
	fmt.Printf("UUID: %s\n", uuid)

	username = readString(packet.Data)
	fmt.Printf("Username: %s\n", username)

	return
}

func writeVarint(w io.Writer, varint int64) error {
	n := binary.PutUvarint(varintBuff[:], uint64(varint))
	_, err := w.Write(varintBuff[0:n])

	checkError(err)

	return err
}

func writeString(w io.Writer, s string) (err error) {
	bs := []byte(s)
	writeVarint(w, int64(len(bs)))

	_, err = w.Write(bs)
	checkError(err)

	return
}

func readString(reader *bytes.Buffer) string {
	stringSize, err := binary.ReadUvarint(reader)
	checkError(err)

	stringbytes := make([]byte, stringSize)
	reader.Read(stringbytes)

	return string(stringbytes)
}

func writeData(w io.Writer, data interface{}) {
	err := binary.Write(w, binary.BigEndian, data)
	checkError(err)
}

func sendPacket(w io.Writer, buf *bytes.Buffer) {
	sendBuffer := bufio.NewWriter(w)

	writeVarint(sendBuffer, int64(len(buf.Bytes())))
	buf.WriteTo(sendBuffer)

	sendBuffer.Flush()
}

func readPacket(reader *bufio.Reader) (packet *Packet) {
	size, err := binary.ReadUvarint(reader)
	if err != nil {
		fmt.Println("Error reading from connection, returning from reciver. Error: " + err.Error())
		return
	}

	// ignore the packet type in the packet length since we read it too
	size -= 1

	packetType, err := binary.ReadUvarint(reader)
	checkError(err)

	data := make([]byte, size)
	n, _ := reader.Read(data)

	for n < int(size) {
		// didn't read all the data, keep trying to read some more
		nx, _ := reader.Read(data[n:])
		n += nx
	}

	return &Packet{Id: int(packetType), Size: int(size), Data: bytes.NewBuffer(data)}
}
