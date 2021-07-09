package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/ngagnon/flybywire/internal/wire"
)

type target struct {
	path string
	host string
}

type knownHost struct {
	host        string
	fingerprint string
}

type fingerprintError struct {
	fingerprint string
	err         string
	changed     bool
}

type remoteFileInfo struct {
	isFile bool
}

func flycp(args []string) {
	f := flag.NewFlagSet("cp", flag.ContinueOnError)
	notls := f.Bool("notls", false, "Disable TLS")

	err := f.Parse(args)

	if err != nil {
		printUsage()
		return
	}

	args = f.Args()

	if len(args) != 2 {
		if len(args) > 2 {
			fmt.Println("Only 2 arguments (with no * wildcard) are supported at this time")
			fmt.Print()
		}

		printUsage()
		return
	}

	source := parseTarget(args[0])
	dest := parseTarget(args[1])

	if source.host != "" && dest.host != "" {
		fmt.Println("Transfers between servers are not currently supported")
		fmt.Println()
		return
	}

	if source.host == "" && dest.host == "" {
		fmt.Println("Local file transfers are not currently supported")
		fmt.Println()
		return
	}

	host := source.host

	if dest.host != "" {
		host = dest.host
	}

	conn, err := connect(host, *notls)

	var e *fingerprintError

	if errors.As(err, &e) {
		if e.changed {
			fmt.Println("REMOTE HOST IDENTIFICATION HAS CHANGED!!!")
			fmt.Println("It is possible that someone is doing something nasty!")
			fmt.Printf("The host fingerprint is %s\n", e.fingerprint)
			fmt.Println("Add this fingerprint to ~/.fly/known_hosts to get rid of this message.")
			return
		}

		if !trustPrompt(host, e.fingerprint) {
			return
		}

		err = allowFingerprint(host, e.fingerprint)

		if err != nil {
			fmt.Printf("Failed to add fingerprint to known hosts: %v\n", err)
			return
		}

		conn, err = connect(host, *notls)
	}

	if err != nil {
		fmt.Printf("Failed to connect to %s: %v\n", host, err)
		return
	}

	defer conn.Close()

	reader := wire.NewReader(conn)

	if source.host == "" {
		upload(conn, reader, source, dest)
	} else {
		download(conn, reader, source, dest)
	}
}

func download(conn net.Conn, reader *wire.WireReader, source target, dest target) {
	info, found := statRemoteFile(conn, reader, source.path)

	if !found {
		log.Fatalln("Remote: No such file or directory")
	}

	if !info.isFile {
		log.Fatalln("Only regular file downloads are currently supported.")
	}

	dstInfo, err := os.Stat(dest.path)

	if err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Fatalf("%s: %v\n", dest.path, err)
	}

	if !errors.Is(err, os.ErrNotExist) && dstInfo.IsDir() {
		dest.path = path.Join(dest.path, path.Base(source.path))
	}

	tmpPath := dest.path + ".fly-download"
	f, err := os.Create(tmpPath)

	if err != nil {
		log.Fatalf("%s: %v\n", dest.path, err)
	}

	r := sendCommand(conn, reader, "STREAM", "R", source.path)

	if wireErr, ok := r.(*wire.Error); ok {
		log.Fatalf("Remote: %s\n", wireErr.Message)
	}

	streamId := strconv.Itoa(r.(*wire.Integer).Value)

	for {
		val, err := reader.Read()

		if err != nil {
			log.Fatalf("Failed to read from socket: %v\n", err)
		}

		tagged, isTagged := val.(*wire.TaggedValue)

		if !isTagged {
			log.Fatalf("Unexpected %s, was expected tag\n", val.Name())
		}

		if tagged.Tag != streamId {
			log.Fatalf("Unexpected stream ID %s\n", tagged.Tag)
		}

		if tagged.Value == wire.Null {
			break
		}

		blob, isBlob := tagged.Value.(*wire.Blob)

		if !isBlob {
			log.Fatalf("Unexpected %s, was expected blob\n", tagged.Value.Name())
		}

		_, err = f.Write(blob.Data)

		if err != nil {
			log.Fatalf("Failed to write to %s: %v\n", dest.path, err)
		}
	}

	f.Close()

	err = os.Rename(tmpPath, dest.path)

	if err != nil {
		log.Fatalf("Rename failed: %v\n", err)
	}
}

func upload(conn net.Conn, reader *wire.WireReader, source target, dest target) {
	info, err := os.Stat(source.path)

	if err != nil {
		log.Fatalf("%s: %v\n", source.path, err)
	}

	if !info.Mode().IsRegular() {
		log.Fatalln("Only regular file uploads are currently supported.")
	}

	f, err := os.Open(source.path)

	if err != nil {
		fmt.Printf("%s: %v\n", source.path, err)
		return
	}

	defer f.Close()

	// When copying to a folder, append the source filename to the destination path
	if info, found := statRemoteFile(conn, reader, dest.path); found && !info.isFile {
		dest.path = path.Join(dest.path, path.Base(source.path))
	}

	r := sendCommand(conn, reader, "STREAM", "W", dest.path)

	if wireErr, ok := r.(*wire.Error); ok {
		log.Fatalf("Remote: %s\n", wireErr.Message)
	}

	streamId := strconv.Itoa(r.(*wire.Integer).Value)
	buf := make([]byte, 32*1024)

	for {
		n, err := f.Read(buf)

		if err == io.EOF {
			break
		}

		if err != nil {
			fmt.Printf("Failed to read from %s: %v\n", source.path, err)
			return
		}

		blob := wire.NewBlob(buf[0:n])
		err = wire.NewTaggedValue(blob, streamId).WriteTo(conn)

		if err != nil {
			fmt.Printf("Failed to write to socket: %v\n", err)
			return
		}
	}

	err = wire.NewTaggedValue(wire.Null, streamId).WriteTo(conn)

	if err != nil {
		fmt.Printf("Failed to write to socket: %v\n", err)
		return
	}

	for i := 0; i < 10; i++ {
		r := sendCommand(conn, reader, "LIST", dest.path)

		if _, isErr := r.(*wire.Error); !isErr {
			return
		}

		time.Sleep(10 * time.Millisecond)
	}

	fmt.Println("Unknown error occurred")
}

func statRemoteFile(conn net.Conn, reader *wire.WireReader, remotePath string) (info remoteFileInfo, found bool) {
	r := sendCommand(conn, reader, "LIST", remotePath)

	wireErr, isErr := r.(*wire.Error)

	if isErr {
		if wireErr.Code == "NOTFOUND" {
			return remoteFileInfo{}, false
		} else {
			log.Fatalf("Remote: %s\n", wireErr.Message)
		}
	}

	table, isTable := r.(*wire.Table)
	fileName := path.Base(remotePath)

	info = remoteFileInfo{}
	info.isFile = fileName != "" &&
		fileName != "/" &&
		fileName != "." &&
		fileName != ".." &&
		isTable &&
		table.RowCount == 1 &&
		table.Row(0)[0].(*wire.String).Value == "F" &&
		table.Row(0)[1].(*wire.String).Value == fileName

	return info, true
}

func trustPrompt(host string, fingerprint string) bool {
	fmt.Printf("The authenticity of host %s cannot be established\n", host)
	fmt.Printf("Host fingerprint is %s\n", fingerprint)

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("Are you sure you want to continue connecting (yes/no)? ")
		line, err := reader.ReadString('\n')

		if err != nil {
			fmt.Printf("Failed to read user input: %v\n", err)
			return false
		}

		line = strings.TrimSuffix(line, "\n")

		if line == "yes" {
			return true
		}

		if line == "no" {
			return false
		}
	}
}

func parseTarget(s string) target {
	if !strings.HasPrefix(s, "//") {
		return target{path: s}
	}

	s = s[2:]
	i := strings.Index(s, "/")
	t := target{}

	if i == -1 {
		t.host = s
		t.path = "/"
	} else {
		t.host = s[0:i]
		t.path = s[i+1:]
	}

	if !strings.Contains(t.host, ":") {
		t.host += ":6767"
	}

	return t
}

func verifyPeerCertificate(host string, rawCert []byte) error {
	knownHosts, err := readKnownHosts()

	if err != nil {
		return fmt.Errorf("failed to read .fly/known_hosts: %w", err)
	}

	fingerprint := fmt.Sprintf("%x", sha256.Sum256(rawCert))

	for _, knownHost := range knownHosts {
		if knownHost.host == host {
			if knownHost.fingerprint == fingerprint {
				return nil
			} else {
				return &fingerprintError{
					fingerprint: fingerprint,
					changed:     true,
					err:         "TLS fingerprint was changed",
				}
			}
		}
	}

	return &fingerprintError{
		fingerprint: fingerprint,
		changed:     false,
		err:         "unknown TLS fingerprint",
	}
}

func readKnownHosts() ([]knownHost, error) {
	homeDir, err := os.UserHomeDir()

	if err != nil {
		return nil, fmt.Errorf("failed to get user home: %w", err)
	}

	hostPath := path.Join(homeDir, ".fly/known_hosts")
	f, err := os.Open(hostPath)

	if errors.Is(err, os.ErrNotExist) {
		return []knownHost{}, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to open .fly/known_hosts for reading: %w", err)
	}

	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()

	if err != nil {
		return nil, fmt.Errorf("failed to read .fly/known_hosts: %w", err)
	}

	knownHosts := make([]knownHost, 0, len(records))

	for _, record := range records {
		knownHosts = append(knownHosts, knownHost{
			host:        record[0],
			fingerprint: record[1],
		})
	}

	return knownHosts, nil
}

func allowFingerprint(host, fingerprint string) error {
	homeDir, err := os.UserHomeDir()

	if err != nil {
		return fmt.Errorf("failed to get user home: %w", err)
	}

	flyFolder := path.Join(homeDir, ".fly")
	_, err = os.Stat(flyFolder)

	if errors.Is(err, os.ErrNotExist) {
		if err = os.MkdirAll(flyFolder, 0700); err != nil {
			return fmt.Errorf("failed to create .fly: %w", err)
		}
	}

	hostPath := path.Join(flyFolder, "known_hosts")
	f, err := os.OpenFile(hostPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)

	if err != nil {
		return fmt.Errorf("failed to open .fly/known_hosts for writing: %w", err)
	}

	defer f.Close()

	writer := csv.NewWriter(f)
	writer.Write([]string{host, fingerprint})
	writer.Flush()

	if err := writer.Error(); err != nil {
		return fmt.Errorf("failed to write to .fly/known_hosts: %w", err)
	}

	return nil
}

func (err *fingerprintError) Error() string {
	return err.err
}
