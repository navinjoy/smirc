// This is a minimalistic IRC server. See: https://www.ietf.org/rfc/rfc1459.txt
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

// --- Web Server Endpoints
const (
	endPointSendMessage           = "/send-message"
	endPointGetMessagesForChannel = "/get-messages-for-channel"
	endPointGetUsersForChannel    = "/get-users-for-channel"
)

// --- HTML Components
const (
	formKeyMessage = "message"
)

// --- Default Config Values
const (
	defaultIRCServer           = "irc.freenode.net"
	defaultIRCPort             = 6667
	defaultWebServerPortNumber = 8080
	defaultChannel             = "#midnightcafe"
)

// --- Environment Variables
var (
	envVarNickName       = os.Getenv("IRC_NICKNAME")
	envVarUserName       = os.Getenv("IRC_USERNAME")
	envVarRealName       = os.Getenv("IRC_REALNAME")
	envVarConfigFileName = os.Getenv("CONFIG_FILENAME")
)

var (
	lastWho = time.Now().Add(-1)
)

// IRCConfig keeps the config needed to connect to the IRC network
type IRCConfig struct {
	Server              string `json:"server"`
	Port                int    `json:"port"`
	Channel             string `json:"channel"`
	WebServerPortNumber int    `json:"web-server-port-number"`
}

// IRC keeps all the inbound and outbound IRC messages
type IRC struct {
	messagesMutex sync.Mutex
	messages      []IRCMessage
	usersMutex    sync.Mutex
	users         map[string]*User
	config        *IRCConfig
	conn          net.Conn
}

// User is an IRC User
type User struct {
	Nickname string
	Hostname string
	Server   string
	Channel  string
}

// IRCMessage is a message sent or received from the IRC network
type IRCMessage struct {
	channel  string
	userName string
	message  string
}

func (irc *IRC) Join() {
	log.Printf(">> JOIN %s\n\n", irc.config.Channel)
	_, _ = fmt.Fprintf(irc.conn, "JOIN %s\r\n", irc.config.Channel)
}

func (irc *IRC) Pong(message string) {
	log.Printf(">> PONG %s\n\n", message[5:])
	_, _ = fmt.Fprintf(irc.conn, "PONG %s\r\n", message[5:])
}

func (irc *IRC) AddIncomingMessage(chatRoom, userName, message string) {
	irc.messagesMutex.Lock()
	defer irc.messagesMutex.Unlock()
	irc.messages = append(irc.messages, IRCMessage{chatRoom, userName, message})
}

func (irc *IRC) SendMessage(chatRoom, message string) {
	irc.messagesMutex.Lock()
	defer irc.messagesMutex.Unlock()
	irc.messages = append(irc.messages, IRCMessage{chatRoom, envVarNickName, message})
	sendMessage(irc.conn, irc.config.Channel, message)
}

func (irc *IRC) GetMessagesForChatRoom(channel string) string {
	irc.messagesMutex.Lock()
	defer irc.messagesMutex.Unlock()
	var msgs []string
	for _, m := range irc.messages {
		if m.channel == channel {
			msgs = append(msgs, fmt.Sprintf("%s: %s", m.userName, m.message))
		}
	}
	return strings.Join(msgs, "<br/>")
}

func (irc *IRC) GetUsersForChannel() string {
	irc.usersMutex.Lock()
	defer irc.usersMutex.Unlock()
	var users []string
	for _, u := range irc.users {
		if u.Channel == irc.config.Channel {
			users = append(users, u.Nickname)
		}
	}
	sort.Strings(users)
	return strings.Join(users, ",")
}

func (irc *IRC) ResetUsersForChannel() {
	irc.usersMutex.Lock()
	defer irc.usersMutex.Unlock()
	irc.users = nil
}

func (irc *IRC) RemoveUser(nickname string) {
	irc.usersMutex.Lock()
	defer irc.usersMutex.Unlock()
	nickname = strings.Trim(nickname, ":@+ \n")
	delete(irc.users, nickname)
}

func (irc *IRC) AddUserForChannel(user *User) {
	irc.usersMutex.Lock()
	defer irc.usersMutex.Unlock()
	// Remove any special characters from the nickname, username, and hostname
	user.Nickname = strings.Trim(user.Nickname, ":@+ \n")
	user.Hostname = strings.Trim(user.Hostname, ":@+ \n")
	irc.users[user.Nickname] = user
}

var irc = &IRC{}

func handlerSendMessage(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		log.Printf("Error: %s", err)
		http.Redirect(w, r, "/", 302)
		return
	}
	message := r.Form.Get(formKeyMessage)
	irc.SendMessage(irc.config.Channel, message)
	http.Redirect(w, r, "/", 302)
}

func handlerGetMessagesForChannel(w http.ResponseWriter, r *http.Request) {
	content := `<!doctype html><html itemscope="" itemtype="http://schema.org/WebPage" lang="en">
	<head><title>minirc: messages</title><meta http-equiv="refresh" content="1"></head>
    <body>` + irc.GetMessagesForChatRoom(irc.config.Channel) + `</body></html>`
	_, _ = fmt.Fprintf(w, "%s", content)
}

func handlerGetUsersForChannel(w http.ResponseWriter, r *http.Request) {
	content := `<!doctype html><html itemscope="" itemtype="http://schema.org/WebPage" lang="en">
	<head><title>minirc: users</title><meta http-equiv="refresh" content="5"></head>
    <body><strong>Users:</strong> ` + irc.GetUsersForChannel() + `</body></html>`
	_, _ = fmt.Fprintf(w, "%s", content)
}

func handlerIndex(w http.ResponseWriter, r *http.Request) {
	content := `<!doctype html><html itemscope="" itemtype="http://schema.org/WebPage" lang="en">
	<head><title>minirc</title></head><body>
      <iframe marginwidth="0" marginheight="0" width="500" height="500" scrolling="no" frameborder=0 src="` + endPointGetMessagesForChannel + `">
      </iframe>
      <iframe marginwidth="0" marginheight="0" width="500" height="25" scrolling="no" frameborder=0 src="` + endPointGetUsersForChannel + `">
      </iframe>
      <form action="` + endPointSendMessage + `">
        <input type="text" id="` + formKeyMessage + `" name="` + formKeyMessage + `" />
        <input type="submit" value="Send" />
      </form></body></html>`
	_, _ = fmt.Fprintf(w, "%s", content)
}

func connectToIRC(irc *IRC) net.Conn {
	if envVarNickName == "" || envVarUserName == "" || envVarRealName == "" {
		log.Fatal("Environment variables IRC_NICKNAME, IRC_USERNAME, IRC_REALNAME are required")
	}
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", irc.config.Server, irc.config.Port))
	if err != nil {
		fmt.Printf("Failed to connect to IRC server [%s:%d]: %s\n", irc.config.Server, irc.config.Port, err)
		return nil
	}

	_, _ = fmt.Fprintf(conn, "USER %s 0 * :realname\r\n", envVarUserName)
	_, _ = fmt.Fprintf(conn, "NICK %s\r\n", envVarNickName)

	reader := bufio.NewReader(conn)

	// Continuously read messages from the server
	go func() {
		for {
			message, err := reader.ReadString('\n')
			if err != nil {
				log.Fatalf("Failed to read message from IRC server: %s\n", err)
			}

			fmt.Print(message)
			irc.AddIncomingMessage("", "", message)
			spaceDelimited := strings.SplitN(message, " ", 3)
			messageCode := spaceDelimited[1]

			if messageCode == "001" {
				irc.Join()
			}

			if message[0:4] == "PING" {
				irc.Pong(message)
			}

			// Message sent to the channel
			if strings.Contains(message, fmt.Sprintf("PRIVMSG %s", irc.config.Channel)) {
				parts := strings.SplitN(message, ":", 3)
				if len(parts) == 3 {
					username := strings.Split(parts[1], "!")[0]
					msg := strings.TrimSpace(parts[2])
					fmt.Printf("[%s] %s: %s\n", irc.config.Channel, username, msg)
					irc.AddIncomingMessage(irc.config.Channel, username, msg)
				}
			}

			// Get Users
			if messageCode == "353" {
				getUsersFrom353(message)
			}

			// Get Users
			if messageCode == "352" {
				getUsersFrom352(message)
			}

			// Send WHO once every 30 seconds to refresh the list
			if time.Since(lastWho) > 30*time.Second {
				// Send a WHO command to the server to get a list of users in the #midnightcafe channel
				_, _ = fmt.Fprintf(conn, "WHO %s\r\n", irc.config.Channel)
				// irc.ResetUsersForChannel()
				lastWho = time.Now()
			}

			if strings.Contains(message, " JOIN ") {
				getUserFromNewJoin(message)
			}

			if strings.Contains(message, " PART ") {
				removeNick(message)
			}
		}
	}()
	return conn
}

func getUserFromNewJoin(message string) {
	// :web-50!web-50@freenode-otsuav.ut8c.4jho.iho72g.IP JOIN :#midnightcafe
	// :<nick>!<user>@host JOIN :<channel>
	parts := strings.Split(message, " ")
	user := &User{
		Nickname: strings.Split(parts[0], "!")[0],
		Channel:  strings.Trim(parts[2], ":"),
	}
	irc.AddUserForChannel(user)
}

func removeNick(message string) {
	// :web-50!web-50@freenode-otsuav.ut8c.4jho.iho72g.IP PART :#midnightcafe
	// :<nick>!<user>@server PART :<channel>
	parts := strings.Split(message, " ")
	nick := strings.Split(parts[0], "!")[0]
	irc.RemoveUser(nick)
}

func getUsersFrom353(message string) {
	// <server>        353 <my-nickname>    = <channel>     :<nick> <nick>
	// :*.freenode.net 353 HelloMyNameIsGNU = #midnightcafe :@web-50 HelloMyNameIsGNU

	parts := strings.Split(message, " ")
	if len(parts) < 5 {
		return
	}

	for idx, user := range parts {
		if idx < 5 {
			continue
		}
		user := &User{
			Nickname: user,
			Channel:  parts[4],
		}
		irc.AddUserForChannel(user)
	}
}

func getUsersFrom352(message string) {
	// The WHO command response has the following format:
	// <server> 352 <my-nickname> <channel> <username> <hostname> <server> <nickname> <H|G>[*][@|+] :<hopcount> <realname>
	// Example:
	// :*.freenode.net 352 HelloMyNameIsGNU #midnightcafe web-50     freenode-otsuav.ut8c.4jho.iho72g.IP *.freenode.net web-50     H@s           :0          https://kiwiirc.com/
	// <server>        352 <my-nickname>    <channel>     <username> <hostname>                          <server>       <nickname> <H|G>[*][@|+] :<hopcount> <realname>

	parts := strings.Split(message, " ")
	if len(parts) < 9 {
		return
	}

	user := &User{
		// Remove any special characters from the nickname, username, and hostname
		Nickname: parts[7],
		Hostname: parts[5],
		Channel:  parts[3],
		Server:   parts[6],
	}
	irc.AddUserForChannel(user)
}

func sendMessage(conn net.Conn, channel string, message string) {
	log.Printf("Sending message: PRIVMSG %s :%s\r\n", channel, message)
	// Send the message to the channel
	_, _ = fmt.Fprintf(conn, "PRIVMSG %s :%s\r\n", channel, message)
}

func readConfig(fileName string) *IRCConfig {
	var config IRCConfig
	// Load the JSON file
	data, err := os.ReadFile(fileName)
	if err != nil {
		log.Fatalf("Failed to read config file [%s]: %s", fileName, err)
	}

	// Parse the JSON into a Config struct
	if err := json.Unmarshal(data, &config); err != nil {
		fmt.Println("Failed to parse config file:", err)
		return nil
	}

	if config.Port == 0 {
		config.Port = defaultIRCPort
	}
	if config.Server == "" {
		config.Server = defaultIRCServer
	}
	if config.WebServerPortNumber == 0 {
		config.WebServerPortNumber = defaultWebServerPortNumber
	}
	if config.Channel == "" {
		config.Channel = defaultChannel
	}

	fmt.Printf("Config: %+v\n", config)
	return &config
}

func main() {
	irc.config = readConfig(envVarConfigFileName)
	irc.conn = connectToIRC(irc)
	irc.users = make(map[string]*User)

	http.HandleFunc("/", handlerIndex)
	http.HandleFunc(endPointGetMessagesForChannel, handlerGetMessagesForChannel)
	http.HandleFunc(endPointGetUsersForChannel, handlerGetUsersForChannel)
	http.HandleFunc(endPointSendMessage, handlerSendMessage)

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", irc.config.WebServerPortNumber), nil))
}
