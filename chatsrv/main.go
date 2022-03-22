package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const debuggingMode bool = true

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

func is_array_found(val string, array []string) (ok bool, i int) {
	for i = range array {
		if ok = array[i] == val; ok {
			return
		}
	}
	return
}

func is_allowed(val string, array []Allow) (ok bool, i int) {
	for i = range array {
		if ok = array[i].IpAddress == val; ok {
			return
		}
	}
	return
}

func remove_slice(slice []string, remove_this string) []string {
	for i, v := range slice {
		if v == remove_this {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}

var channel chan string // Connections Channel
var my_hosts []string   // Names of connected Hosts

type AllEventConnections struct {
	details map[string][]chan string
}

var AEventConnection AllEventConnections

// AddHandler adds an event listener to the AllMyConnections struct instance
func (b *AllEventConnections) AddHandler(e string, ch chan string) {
	if b.details == nil {
		b.details = make(map[string][]chan string)
	}
	if _, ok := b.details[e]; ok {
		b.details[e] = append(b.details[e], ch)
	} else {
		b.details[e] = []chan string{ch}
	}
}

// RemoveHandler removes an event listener from the AllMyConnections struct instance
func (b *AllEventConnections) RemoveHandler(e string, ch chan string) {
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
func (b *AllEventConnections) Emit(e string, response string) {
	if _, ok := b.details[e]; ok {
		for _, handler := range b.details[e] {
			go func(handler chan string) {
				handler <- response
			}(handler)
		}
	}
}

func is_offline(msg string, hostName string) bool {
	s := strings.SplitN(msg, ":", 2) // Seperate host from Message
	theHost := s[0]
	theMessage := s[1]
	if theHost == hostName && (strings.Contains(theMessage, "I'm Offline at") || strings.Contains(theMessage, "I'm out...")) {
		return true
	}
	return false
}

func retry(c net.Conn, msg string, hostName string, remotePC string) bool {
	time.Sleep(time.Minute)
	i, e := c.Write([]byte(msg))
	if i > 0 && e == nil {
		fmt.Printf("Retry %s was able to Finally the send MSG to => %s", remotePC, msg)
		return true
	} else {
		fmt.Printf("FAILED %s to Send MSG to => %s", remotePC, msg)
		deleteConnection(c, hostName, remotePC)
		return false
	}
}

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

	AEventConnection = AllEventConnections{nil}
	channel = make(chan string)

	// run loop forever, until exit.
	for {
		// Listen/Wait for an incoming connection.
		c, err := l.Accept()
		if err != nil {
			fmt.Println("Error connecting:", err.Error())
			return
		}
		fmt.Println("Client connected.")

		remotePC := c.RemoteAddr().String()
		remoteIP := strings.Split(remotePC, ":") // Needs work for IPv6 Support, I'm guessing...here as delimiting on :
		// Print client connection address.
		fmt.Println("Client " + remotePC + " connected.")

		allowed := SocketConn.Hosts.Allowed                // Get allowed connections from YAML config
		success, index := is_allowed(remoteIP[0], allowed) // Check if on allowed list from YAML config
		if success {
			hostName := SocketConn.Hosts.Allowed[index].HostName // What did you name the IP connection in YAML config
			my_hosts = append(my_hosts, remotePC)

			AEventConnection.AddHandler(remotePC, channel) // Register hostName talking and channel

			// Loop through Emits on channel
			go func() {
				for {
					msg := <-channel
					i, e := c.Write([]byte(msg))
					if i > 0 && e == nil {
						fmt.Printf("%s has Got MSG => %s", remotePC, msg)
					} else {
						if !is_offline(msg, hostName) {
							fmt.Printf("%s was UNable to Send MSG to => %s", remotePC, msg)
							go retry(c, msg, hostName, remotePC)
							break // Bail, issues
						} else {
							deleteConnection(c, hostName, remotePC)
							break // Bail, issues
						}
					}
				}
			}()

			// Handle connections concurrently in a new goroutine.
			go handleConnection(c, hostName, remotePC)
		} else {
			fmt.Println("Termanated unKnown Client " + remotePC + " not on Allowed List!!")
		}
	}
}

func talk(buffer []byte, hostName string) {
	b := []byte(hostName + ":")
	data := append(b, buffer...)
	s := string(data)
	checkForNL := s[len(s)-1:]
	if checkForNL != "\n" {
		s += "\n" // Let's be safe...and add the Delimiting New Line, if not Found!
	}

	// Send response message to the clients.
	for _, key := range my_hosts {
		if debuggingMode {
			log.Printf("Emitting Message to: %s => %s", key, s)
		}
		AEventConnection.Emit(key, s)
	}
}

var lastDisconnected string

func deleteConnection(conn net.Conn, hostName string, remotePC string) {
	if lastDisconnected == remotePC {
		checker, index := is_array_found(remotePC, my_hosts)
		if checker {
			log.Fatal("Still Found Sclice @ " + string(index))
		}
		return
	}

	lastDisconnected = remotePC
	AEventConnection.RemoveHandler(remotePC, channel)

	my_hosts = remove_slice(my_hosts, remotePC)

	leftMsg := "I'm out...\n"
	b := []byte(leftMsg) // Include Delimiter LiNe feed...(\n)
	fmt.Println(leftMsg)
	talk(b, hostName)
	conn.Close()
}

// handleConnection handles logic for a single connection request.
func handleConnection(conn net.Conn, hostName string, remotePC string) {
	// Buffer client input until a newline.
	buffer, err := bufio.NewReader(conn).ReadBytes('\n') // Should include New Line...

	// Close left clients.
	if err != nil {
		deleteConnection(conn, hostName, remotePC)
		return
	}

	talk(buffer, hostName)

	// Restart the process.
	handleConnection(conn, hostName, remotePC)
}
