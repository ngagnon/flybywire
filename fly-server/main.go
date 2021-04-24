package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
)

type reader struct {
	reader io.Reader
}

type writer struct {
	writer io.Writer
}

type session struct {
	terminated bool
	user       string
	writer     *writer
}

type user struct {
	username string
	password []byte
	chroot   string
	admin    bool
}

type access rune

const (
	Denied access = '_'
	Read   access = 'R'
	Write  access = 'W'
)

type acp struct {
	name       string
	users      []string
	paths      []string
	fileAccess access
	acpAccess  access
}

type commandHandler func([]string, *session)

type command struct {
	handler commandHandler
	args    []string
}

var commandHandlers = map[string]commandHandler{
	"PING":    handlePing,
	"QUIT":    handleQuit,
	"MKDIR":   handleMkdir,
	"ADDUSER": handleAddUser,
}

var dir string
var singleUser bool
var users map[string]user
var policies []acp
var globalLock sync.RWMutex

// @TODO: each command should have a unit test to make sure it calls checkAuth,
// and to make sure it returns -DENIED when checkAuth returned false
// @TODO: protocol should just use single line feeds
// @TODO: handle invalid inputs in the protocol
// @TODO: custom config path
// @TODO: should allow you to pass a single file instead of a dir
func main() {
	port := flag.Int("port", 6767, "TCP port to listen on")
	flag.Parse()

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))

	if err != nil {
		fmt.Println("Cannot start TCP server:", err)
		os.Exit(1)
	}

	defer ln.Close()

	if flag.NArg() == 0 {
		fmt.Println("USAGE: fly-server ROOTDIR")
		fmt.Println()
		os.Exit(1)
	}

	dir = flag.Arg(0)
	readDatabase()

	for {
		conn, err := ln.Accept()

		if err != nil {
			fmt.Println("Accept error")
			os.Exit(1)
		}

		go handleSession(conn)
	}
}

func handleSession(conn net.Conn) {
	connReader := &reader{reader: conn}
	r := bufio.NewReader(connReader)

	defer conn.Close()

	session := session{
		terminated: false,
		writer:     &writer{writer: conn},
	}

	defer func() {
		if err := recover(); err != nil && err != io.EOF {
			fmt.Println("ERROR: handleSession:", err)
			fmt.Println(string(debug.Stack()))
		}
	}()

	for !session.terminated {
		cmd := parseCommand(r, session.writer)
		cmd.handler(cmd.args, &session)
	}
}

func parseCommand(r *bufio.Reader, writer *writer) command {
	var line []byte
	var err error

	for {
		for {
			line = readLine(r)

			if len(line) > 0 {
				break
			}
		}

		if bytes.HasPrefix(line, []byte("*")) {
			var n int

			line = bytes.TrimPrefix(line, []byte("*"))

			if n, err = strconv.Atoi(string(line)); err != nil {
				panic(err)
			}

			arr := make([]string, n)

			for i := 0; i < n; i++ {
				line = readLine(r)
				line = bytes.TrimPrefix(line, []byte("$"))

				var len int

				if len, err = strconv.Atoi(string(line)); err != nil {
					panic(err)
				}

				var b1, b2 byte

				buf := make([]byte, len)
				io.ReadFull(r, buf)
				b1, _ = r.ReadByte()
				b2, _ = r.ReadByte()

				if b1 != '\r' || b2 != '\n' {
					// @TODO: return syntax error
				}

				arr[i] = string(buf)
			}

			if handler, ok := getCommandHandler(arr[0]); ok {
				return command{
					handler: handler,
					args:    arr[1:],
				}
			} else {
				fmt.Fprintf(writer, "-ERR Unknown command '%s'\r\n", arr[0])
			}
		} else {
			fields := strings.Fields(string(line))

			if len(fields) > 0 {
				if handler, ok := getCommandHandler(fields[0]); ok {
					return command{
						handler: handler,
						args:    fields[1:],
					}
				} else {
					fmt.Fprintf(writer, "-ERR Unknown command '%s'\r\n", fields[0])
				}
			} else {
				fmt.Fprint(writer, "-ERR Protocol error\r\n")
			}
		}
	}
}

func readLine(r *bufio.Reader) (line []byte) {
	for {
		var data []byte

		data, _ = r.ReadBytes('\n')
		line = append(line, data...)

		if bytes.HasSuffix(line, []byte("\r\n")) {
			break
		}
	}

	line = bytes.TrimSuffix(line, []byte("\r\n"))
	return
}

func (w *writer) Write(p []byte) (int, error) {
	n, err := w.writer.Write(p)

	if err != nil {
		panic(err)
	}

	return n, nil
}

func (r *reader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)

	if err != nil {
		panic(err)
	}

	return n, nil
}

func getCommandHandler(s string) (h commandHandler, ok bool) {
	h, ok = commandHandlers[strings.ToUpper(s)]
	return
}
