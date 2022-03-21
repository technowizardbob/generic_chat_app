package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Allow struct {
	IpAddress string `yaml:"ip"`
	HostName  string `yaml:"host"`
}

type ChatData struct {
	Conf struct {
		ConnHost string `yaml:"host"`
		ConnPort string `yaml:"port"`
		ConnType string `yaml:"type"`
	}
	Hosts struct {
		Allowed []Allow `yaml:"allowed"`
	}
}

func readConf(filename string) (*ChatData, error) {
	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	c := &ChatData{}
	err = yaml.Unmarshal(buf, c)
	if err != nil {
		return nil, fmt.Errorf("in file %q: %v", filename, err)
	}

	return c, nil
}

func is_allowed(val string, array []Allow) (ok bool, i int) {
	for i = range array {
		if ok = array[i].IpAddress == val; ok {
			return
		}
	}
	return
}

func remove_slice(s []string, r string) []string {
	for i, v := range s {
		if v == r {
			return append(s[:i], s[i+1:]...)
		}
	}
	return s
}

type Person struct {
	name    string
	details map[string][]chan string
}

var person Person

// AddHandler adds an event listener to the Person struct instance
func (b *Person) AddHandler(e string, ch chan string) {
	if b.details == nil {
		b.details = make(map[string][]chan string)
	}
	if _, ok := b.details[e]; ok {
		b.details[e] = append(b.details[e], ch)
	} else {
		b.details[e] = []chan string{ch}
	}
}

// RemoveHandler removes an event listener from the Person struct instance
func (b *Person) RemoveHandler(e string, ch chan string) {
	if _, ok := b.details[e]; ok {
		for i := range b.details[e] {
			if b.details[e][i] == ch {
				b.details[e] = append(b.details[e][:i], b.details[e][i+1:]...)
				break
			}
		}
	}
}

// Emit emits an event on the Person struct instance
func (b *Person) Emit(e string, response string) {
	if _, ok := b.details[e]; ok {
		for _, handler := range b.details[e] {
			go func(handler chan string) {
				handler <- response
			}(handler)
		}
	}
}

var channel chan string
var my_hosts []string

func main() {
	SocketConn, cerr := readConf("conf.yaml")
	if cerr != nil {
		log.Fatal(cerr)
	}
	// Start the server and listen for incoming connections.
	fmt.Println("Starting " + SocketConn.Conf.ConnType + " server on " + SocketConn.Conf.ConnHost + ":" + SocketConn.Conf.ConnPort)
	l, err := net.Listen(SocketConn.Conf.ConnType, SocketConn.Conf.ConnHost+":"+SocketConn.Conf.ConnPort)
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}
	// Close the listener when the application closes.
	defer l.Close()

	person = Person{"Me", nil}
	channel = make(chan string)

	// run loop forever, until exit.
	for {
		// Listen for an incoming connection.
		c, err := l.Accept()
		if err != nil {
			fmt.Println("Error connecting:", err.Error())
			return
		}
		fmt.Println("Client connected.")

		remotePC := c.RemoteAddr().String()
		remoteIP := strings.Split(c.RemoteAddr().String(), ":")
		// Print client connection address.
		fmt.Println("Client " + remotePC + " connected.")

		allowed := SocketConn.Hosts.Allowed
		success, index := is_allowed(remoteIP[0], allowed)
		if success {
			hostName := SocketConn.Hosts.Allowed[index].HostName

			my_hosts = append(my_hosts, hostName)
			person.AddHandler(hostName, channel) // Register talk and channel

			go func() {
				for {
					msg := <-channel
					c.Write([]byte(msg))
				}
			}()

			// Handle connections concurrently in a new goroutine.
			go handleConnection(c, hostName)
		} else {
			fmt.Println("Termanated Client " + remotePC + " !!")
		}
	}
	// person.RemoveHandler("talk", channel)
}

// handleConnection handles logic for a single connection request.
func handleConnection(conn net.Conn, hostName string) {
	// Buffer client input until a newline.
	buffer, err := bufio.NewReader(conn).ReadBytes('\n')

	// Close left clients.
	if err != nil {
		fmt.Println("Client left.")
		conn.Close()
		my_hosts = remove_slice(my_hosts, hostName)
		return
	}

	// Print response message, stripping newline character.
	log.Println("Client message:", string(buffer[:len(buffer)-1]))

	b := []byte(hostName + ":")
	data := append(b, buffer...)

	// Send response message to the client.
	for _, key := range my_hosts {
		person.Emit(key, string(data))
	}
	// Restart the process.
	handleConnection(conn, hostName)
}
