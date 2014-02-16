package main

import (
	"bufio"
	"bytes"
	"compress/flate"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
)

type Client struct {
	Host     string
	Port     int
	conn     *net.Conn
	reader   *bufio.Reader
	Uuid     string
	Username string
	packets  chan *Packet
}

type Packet struct {
	Id   int
	Size int
	Data *bytes.Buffer
}

func Ping(host string, port int) (string, error) {
	client := Client{Host: host, Port: port}

	fmt.Printf("Going to ping server %s:%d\n", host, port)

	err := client.connect()
	if err != nil {
		return "", err
	}

	defer client.disconnect()

	client.SendHandshake(1)
	client.SendStatusRequest()

	packet, err := readPacket(client.reader)
	if err != nil {
		return "", err
	}

	return ParseStatusRequest(packet)
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

func JoinServer(host string, port int) (*Client, error) {
	client := Client{Host: host, Port: port}
	fmt.Printf("Going to join server %s:%s\n", host, port)

	err := client.connect()
	if err != nil {
		return nil, err
	}

	client.SendHandshake(2)
	client.SendLoginStart()

	packet, err := readPacket(client.reader)
	if err != nil {
		(*client.conn).Close()
		return nil, err
	}

	client.Username, client.Uuid, err = ParseLoginSuccess(packet)

	if err != nil {
		(*client.conn).Close()
		return nil, err
	}

	client.packets = make(chan *Packet)
	go client.packetReader()

	return &client, nil
}

func (client *Client) packetReader() {
	for {
		packet, err := readPacket(client.reader)

		if err != nil {
			fmt.Printf("ERROR: couldn't read packet: %s\n", err.Error())
		}

		client.packets <- packet
	}
}

func (client *Client) Shutdown() {
	(*client.conn).Close()
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

func (client *Client) SendLoginStart() {
	buf := new(bytes.Buffer)

	writeVarint(buf, 0)             // packet id
	writeString(buf, "gophercraft") // username

	sendPacket(*client.conn, buf)
}

func (client *Client) SendKeepAlive(keepalive int32) {
	buf := new(bytes.Buffer)

	writeVarint(buf, 0)       // packet id
	writeData(buf, keepalive) // keepalive

	sendPacket(*client.conn, buf)
}

func ParseStatusRequest(packet *Packet) (string, error) {
	return readString(packet.Data), nil
}

func ParseLoginSuccess(packet *Packet) (username string, uuid string, err error) {
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

func ParseKeepalive(packet *Packet) (int32, error) {
	var keepalive int32
	err := binary.Read(packet.Data, binary.BigEndian, &keepalive)

	if err != nil {
		return 0, err
	}

	return keepalive, nil
}

func ParseMapChunkBulk(packet *Packet) error {
	var chunkColumnCount int16
	var dataLength int32
	var skyLightSent bool

	binary.Read(packet.Data, binary.BigEndian, &chunkColumnCount)
	binary.Read(packet.Data, binary.BigEndian, &dataLength)
	binary.Read(packet.Data, binary.BigEndian, &skyLightSent)

	fmt.Printf("\n\n** Map Chunk: column count: %d, dataLenght: %d, skyLightSent: %s", chunkColumnCount, dataLength, boolToString(skyLightSent))

	compressedData := make([]byte, dataLength)
	var n int32 = 0

	for n < dataLength {
		nx, _ := packet.Data.Read(compressedData[n:])
		n += int32(nx)
	}

	fmt.Println("Going to try to deflate")

	closer := flate.NewReader(bytes.NewBuffer(compressedData))

	buffer := make([]byte, 2048)
	nc, err := closer.Read(buffer)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Deflated: %d", buffer)

	for nc > 0 {
		nc, err = closer.Read(buffer)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Deflated: %d", buffer)
	}

	fmt.Println("Deflate done.")

	closer.Close()

	return nil
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

func readPacket(reader *bufio.Reader) (*Packet, error) {
	size, err := binary.ReadUvarint(reader)
	if err != nil {
		fmt.Println("Error reading from connection, returning from reciver. Error: " + err.Error())
		return nil, err
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

	return &Packet{Id: int(packetType), Size: int(size), Data: bytes.NewBuffer(data)}, nil
}

func boolToString(what bool) string {
	if what {
		return "True"
	} else {
		return "False"
	}
}
