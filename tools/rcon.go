package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"regexp"
	"strconv"
	"time"
)

// Minimal Source RCON client (enough to flush/save and query players).
const (
	rconAuth    = 3
	rconExecCmd = 2
	rconRespVal = 0
)

type rconConn struct {
	conn net.Conn
}

func rconDial(addr, password string) (*rconConn, error) {
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return nil, err
	}
	rc := &rconConn{conn: conn}
	id, _, err := rc.roundtrip(rconAuth, password)
	if err != nil {
		rc.Close()
		return nil, err
	}
	if id == -1 {
		rc.Close()
		return nil, fmt.Errorf("rcon auth failed (wrong password?)")
	}
	return rc, nil
}

func (rc *rconConn) Close() error { return rc.conn.Close() }

// exec runs a command and returns the server's text response.
func (rc *rconConn) exec(cmd string) (string, error) {
	_, body, err := rc.roundtrip(rconExecCmd, cmd)
	return body, err
}

func (rc *rconConn) roundtrip(typ int32, body string) (int32, string, error) {
	const reqID int32 = 1
	if err := rc.write(reqID, typ, body); err != nil {
		return 0, "", err
	}
	return rc.read()
}

func (rc *rconConn) write(id, typ int32, body string) error {
	payload := new(bytes.Buffer)
	binary.Write(payload, binary.LittleEndian, id)
	binary.Write(payload, binary.LittleEndian, typ)
	payload.WriteString(body)
	payload.WriteByte(0)
	payload.WriteByte(0)

	rc.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	pkt := new(bytes.Buffer)
	binary.Write(pkt, binary.LittleEndian, int32(payload.Len()))
	pkt.Write(payload.Bytes())
	_, err := rc.conn.Write(pkt.Bytes())
	return err
}

func (rc *rconConn) read() (int32, string, error) {
	rc.conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	var length int32
	if err := binary.Read(rc.conn, binary.LittleEndian, &length); err != nil {
		return 0, "", err
	}
	if length < 10 || length > 4096+12 {
		return 0, "", fmt.Errorf("rcon: invalid packet length %d", length)
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(rc.conn, buf); err != nil {
		return 0, "", err
	}
	id := int32(binary.LittleEndian.Uint32(buf[0:4]))
	// buf[4:8] = type; body is the rest minus the two trailing nulls.
	body := string(bytes.TrimRight(buf[8:], "\x00"))
	return id, body, nil
}

var playerCountRe = regexp.MustCompile(`There are (\d+)`)

// rconPlayerCount opens a connection, runs "list", and returns the online count.
func rconPlayerCount(addr, password string) (int, error) {
	rc, err := rconDial(addr, password)
	if err != nil {
		return 0, err
	}
	defer rc.Close()
	resp, err := rc.exec("list")
	if err != nil {
		return 0, err
	}
	m := playerCountRe.FindStringSubmatch(resp)
	if m == nil {
		return 0, fmt.Errorf("rcon: unexpected list response: %q", resp)
	}
	return strconv.Atoi(m[1])
}
