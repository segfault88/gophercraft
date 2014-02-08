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
	conn, err := net.Dial("tcp", host+":"+strconv.Itoa(port))
	checkError(err)

	defer conn.Close()

	json := pingServer(conn, host, port)

	fmt.Printf("Ping Response:\n%s\n", json)
}

func pingServer(conn net.Conn, host string, port int) string {
	sendHandshake(conn, host, port, 1)
	sendStatusRequest(conn)
	return readStatusRequest(conn)
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

func readStatusRequest(conn net.Conn) string {
	reader := bufio.NewReader(conn)
	var err error

	_, err = binary.ReadUvarint(reader)
	checkError(err)

	_, err = binary.ReadUvarint(reader)
	checkError(err)

	var stringSize uint64
	stringSize, err = binary.ReadUvarint(reader)
	checkError(err)

	stringbytes := make([]byte, stringSize)
	reader.Read(stringbytes)

	return string(stringbytes)
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
