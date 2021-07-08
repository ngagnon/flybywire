package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	switch os.Args[1] {
	case "cp":
		flycp(os.Args[2:])
	default:
		printUsage()
	}
}

func printUsage() {
	fmt.Println("Usage: fly cp SOURCE DEST")
	fmt.Println()

	fmt.Println("Pass -notls flag to disable TLS")
	fmt.Println()

	fmt.Println("A path that starts with '//' denotes a remote path e.g. '//host:port/some/path/file.txt'")
}
