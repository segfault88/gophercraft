package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	// "math"
	"net"
	// "os"
	// "strings"
	// "encoding/hex"
	"strconv"
)

var (
	varintBuff [binary.MaxVarintLen64]byte
)

func main() {
	fmt.Println("Gophercraft!\n")

	host := "localhost"
	port := 25565

	json := pingServer(host, port)
	fmt.Printf("Ping Response:\n%s\n\n", json)

	joinServer(host, port)
}

func connect(host string, port int) net.Conn {
	conn, err := net.Dial("tcp", host+":"+strconv.Itoa(port))
	checkError(err)

	return conn
}

func pingServer(host string, port int) string {
	conn := connect(host, port)
	defer conn.Close()

	sendHandshake(conn, host, port, 1)
	sendStatusRequest(conn)
	return readStatusRequest(conn)
}

func joinServer(host string, port int) {
	conn := connect(host, port)
	defer conn.Close()

	fmt.Println("Going to join server")

	sendHandshake(conn, host, port, 2)
	sendLoginStart(conn)

	reader := bufio.NewReader(conn)
	readLoginSuccess(reader)
}

func sendHandshake(conn net.Conn, host string, port int, state int) {
	buf := new(bytes.Buffer)

	writeVarint(buf, 0)            // packet id
	writeVarint(buf, 4)            // protocol version
	writeString(buf, host)         // server address
	writeData(buf, uint16(port))   // server port
	writeVarint(buf, int64(state)) // next state 1 for status, 2 for login

	sendPacket(conn, buf)
}

func sendStatusRequest(conn net.Conn) {
	buf := new(bytes.Buffer)

	writeVarint(buf, 0) // packet id

	sendPacket(conn, buf)
}

func sendLoginStart(conn net.Conn) {
	buf := new(bytes.Buffer)

	writeVarint(buf, 0)             // packet id
	writeString(buf, "gophercraft") // username

	sendPacket(conn, buf)
}

func readStatusRequest(conn net.Conn) string {
	reader := bufio.NewReader(conn)
	var err error

	_, err = binary.ReadUvarint(reader)
	checkError(err)

	_, err = binary.ReadUvarint(reader)
	checkError(err)

	return readString(reader)
}

func readLoginSuccess(reader *bufio.Reader) {
	var err error

	_, err = binary.ReadUvarint(reader)
	checkError(err)

	var packetId uint64
	packetId, err = binary.ReadUvarint(reader)

	if packetId != 2 {
		panic("Expected packet id 2 (Login Success), got: " + strconv.Itoa(int(packetId)))
	}

	fmt.Println("Got login success packet!")

	uuid := readString(reader)
	fmt.Printf("UUID: %s\n", uuid)

	username := readString(reader)
	fmt.Printf("Username: %s\n", username)
}

func checkError(err error) {
	if err != nil {
		panic(err.Error())
	}
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

func readString(reader *bufio.Reader) string {
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
