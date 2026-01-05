package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"time"
)

const (
	port = ":1337"
)

var contacts = []struct {
	Num  int
	Name string
	URL  string
}{
	{1, "Twitter/X", "x.com/hitto_kun"},
	{2, "GitHub", "github.com/hitto-hub"},
	{3, "Zenn", "zenn.dev/hitto"},
	{4, "Qiita", "qiita.com/hitto"},
	{5, "Blog", "hitto-kun.hatenablog.com"},
}

const banner = `
 _     _ _   _
| |__ (_) |_| |_ ___
| '_ \| | __| __/ _ \
| | | | | |_| || (_) |
|_| |_|_|\__|\__\___/

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Welcome to hitto's contact server
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

  Available endpoints:

`

func main() {
	listener, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
	defer listener.Close()

	log.Printf("Contact server listening on %s", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(5 * time.Minute))

	// Send banner
	fmt.Fprint(conn, banner)

	// List contacts
	for _, c := range contacts {
		fmt.Fprintf(conn, "  [%d] %-10s → %s\n", c.Num, c.Name, c.URL)
	}

	fmt.Fprint(conn, "\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")
	fmt.Fprint(conn, "> Select [1-5] or 'q' to quit: ")

	reader := bufio.NewReader(conn)

	for {
		input, err := reader.ReadString('\n')
		if err != nil {
			return
		}

		input = strings.TrimSpace(input)

		if input == "q" || input == "quit" || input == "exit" {
			fmt.Fprint(conn, "\nConnection closed. See you! 👋\n")
			return
		}

		var num int
		if _, err := fmt.Sscanf(input, "%d", &num); err == nil {
			if num >= 1 && num <= len(contacts) {
				c := contacts[num-1]
				fmt.Fprintf(conn, "\n→ Opening %s: https://%s\n\n", c.Name, c.URL)
				fmt.Fprint(conn, "> Select [1-5] or 'q' to quit: ")
				continue
			}
		}

		fmt.Fprint(conn, "Invalid input. Select [1-5] or 'q' to quit: ")
	}
}
