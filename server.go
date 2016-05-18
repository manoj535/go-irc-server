package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	rplWelcomeCode    = "001"
	rplYourHostCode   = "002"
	rplCreatedCode    = "003"
	rplMyinfoCode     = "004"
	errNoMotdCode     = "422"
	errNicknameInUse  = "433"
	rplWhoReplyCode   = "352"
	rplEndOfWhoCode   = "315"
	rplNamReplyCode   = "353"
	rplEndOfNamesCode = "366"
	userModes         = "aio"
	channelModes      = "beIikntPpTl"
	joinCommand       = "JOIN"
	messageCommand    = "MSG"
	partCommand       = "PART"
	userCommand       = "USER"
	nickCommand       = "NICK"
	whoCommand        = "WHO"
	privMsgCommand    = "PRIVMSG"
	serverName        = "irc.example.com"
	crlf              = "\r\n"
)

var clients = make(map[*Client]bool)
var rooms = make(map[*Room]bool)
var mutex = &sync.Mutex{}

type handleCommand func([]string, *Command)

var commandMap = map[string]handleCommand{
	"JOIN":    handleJoinCommand,
	"PART":    handlePartCommand,
	"USER":    handleUserCommand,
	"NICK":    handleNickCommand,
	"WHO":     handleWhoCommand,
	"PRIVMSG": handlePrivateMessageCommand,
}

type Command struct {
	name              string
	client            *Client
	handleJoinCommand ([]string)
}

type Room struct {
	name    string
	clients map[*Client]bool
}

type Client struct {
	conn     net.Conn
	rooms    map[*Room]bool
	nickname string
	username string
	realname string
	hostname string
}

/*type Server struct {
	clients         map[*Client]bool
	channels        map[*Channel]bool
	commandChan    chan *Command
	connectionChan chan net.Conn
	name            string
}*/

func handleJoinCommand(parameters []string, command *Command) {

	if len(parameters) == 0 {
		return
	}
	roomName := parameters[0]
	if strings.ContainsAny(roomName, "#") {
		roomName = roomName[1:]
	}
	room := getRoomFromName(roomName)
	if room == nil {
		room = &Room{name: roomName, clients: make(map[*Client]bool)}
	}
	room.clients[command.client] = true
	mutex.Lock()
	rooms[room] = true
	mutex.Unlock()
	replyJoinCommand(command.client, room)
}

func handlePartCommand(parameters []string, command *Command) {
	if len(parameters) == 0 {
		return
	}

	roomName := parameters[0][1:]
	room := getRoomFromName(roomName)
	if room != nil {
		message := fmt.Sprintf(":%s!%s@%s PART #%s :%s", command.client.nickname,
			command.client.username, command.client.hostname, roomName,
			command.client.nickname)
		for roomClient := range room.clients {
			fmt.Println(message)
			roomClient.conn.Write([]byte(message + "\r\n"))
		}
		delete(room.clients, command.client)
	}
}

func handleUserCommand(parameters []string, command *Command) {

	if len(parameters) != 4 {
		command.client.conn.Write([]byte("Invalid syntax\n"))
		return
	}

	fmt.Println("handleUserCommand")
	username := parameters[0]
	hostname := parameters[2]
	realname := parameters[3][1:]
	command.client.username = username
	command.client.realname = realname
	command.client.hostname = hostname
	//replyNickAndUserCommand(command.client)
}

func handleNickCommand(parameters []string, command *Command) {

	if len(parameters) != 1 {
		command.client.conn.Write([]byte("Invalid syntax\n"))
		return
	}

	nickname := parameters[0]
	clientTest := getClientFromName(nickname)
	if clientTest == nil {
		command.client.nickname = nickname
		replyNickAndUserCommand(command.client)
	} else {
		message := fmt.Sprintf(":%s %s * %s :Nickname is already in use", serverName,
			errNicknameInUse, nickname)
		command.client.conn.Write([]byte(message + crlf))
	}
	fmt.Println("handleNickCommand")

}

func handleWhoCommand(parameters []string, command *Command) {

	if len(parameters) != 1 {
		return
	}

	roomName := parameters[0][1:]
	replyWhoCommand(command.client, roomName)
}

func handlePrivateMessageCommand(parameters []string, command *Command) {

	if len(parameters) < 1 {
		return
	}
	message := parameters[1][1:]
	for i := 2; i < len(parameters); i++ {
		message = message + " " + parameters[i]
	}
	if parameters[0][0] == '#' {
		// Message to room
		roomName := parameters[0][1:]
		finalMessage := ""
		room := getRoomFromName(roomName)
		if room != nil {
			_, present := room.clients[command.client]
			if present {
				for client := range room.clients {
					if strings.Compare(command.client.nickname, client.nickname) != 0 {
						finalMessage = fmt.Sprintf(":%s!%s@%s PRIVMSG #%s :%s",
							command.client.nickname, command.client.username,
							command.client.hostname, roomName, message)
						fmt.Println(finalMessage)
						client.conn.Write([]byte(finalMessage + crlf))
					}
				}
			} else {
				command.client.conn.Write([]byte("Not part of this channel\n"))
			}
		}
	} else {
		// Message to User
		clientName := parameters[0]
		client := getClientFromName(clientName)
		if client != nil {
			message = fmt.Sprintf(":%s!%s@%s PRIVMSG %s :%s", command.client.nickname,
				command.client.username, command.client.hostname, clientName, message)
			fmt.Println(message)
			client.conn.Write([]byte(message + crlf))
		}
	}
}

func handleInvalidCommand(command *Command) {
	command.client.conn.Write([]byte("Invalid commmand\n"))
}

func replyJoinCommand(client *Client, room *Room) {

	message := ""
	//client.nickname = "manoj"
	for roomClient := range room.clients {
		message = fmt.Sprintf(":%s!%s@%s %s #%s", client.nickname, client.username,
			client.hostname, "JOIN", room.name)
		fmt.Println(message)
		roomClient.conn.Write([]byte(message + crlf))
	}

	//message = fmt.Sprintf("%s %s #%s :%s", ":127.0.0.1", "332", channel_name, "test topic")
	//client.conn.Write([]byte(message + crlf))
	//message = fmt.Sprintf("%s %s %s = #%s :@%s", ":127.0.0.1", "353", client.nickname, channel_name, client.nickname)
	message = fmt.Sprintf(":%s %s %s = #%s :@", serverName, rplNamReplyCode,
		client.nickname, room.name)

	for roomClient := range room.clients {
		message = message + roomClient.nickname + " "
	}
	message = strings.TrimRight(message, " ")
	fmt.Println(message)
	client.conn.Write([]byte(message + crlf))
	message = fmt.Sprintf(":%s %s %s #%s :%s", serverName, rplEndOfNamesCode,
		client.nickname, room.name, "End of NAMES list")
	fmt.Println(message)
	client.conn.Write([]byte(message + crlf))

}

func replyNickAndUserCommand(client *Client) {

	nick := ""
	if client.nickname == "" {
		nick = "*"
	} else {
		nick = client.nickname
	}
	//nick = "manoj"
	// send RPL_WELCOME
	message := fmt.Sprintf(":%s %s %s %s", serverName, rplWelcomeCode,
		nick, ":Welcome to the Internet Relay Network ")
	fmt.Println("replyNickandUserCommand:", message)
	client.conn.Write([]byte(message + crlf))
	// send RPL_YOURHOST
	message = fmt.Sprintf(":%s %s %s %s", serverName, rplYourHostCode,
		nick, ":Your host is irc.example.com")
	fmt.Println("replyNickandUserCommand:", message)
	client.conn.Write([]byte(message + crlf))
	// send RPL_CREATED
	message = fmt.Sprintf(":%s %s %s %s", serverName, rplCreatedCode,
		nick, ":This server was created at")
	fmt.Println("replyNickandUserCommand:", message)
	client.conn.Write([]byte(message + crlf))
	// send RPL_MYINFO
	message = fmt.Sprintf(":%s %s %s %s %s %s %s", serverName, rplMyinfoCode,
		nick, "localhost", "1.0", userModes, channelModes)
	fmt.Println("replyNickandUserCommand:", message)
	client.conn.Write([]byte(message + crlf))
	// send ERR_NOMOTD
	message = fmt.Sprintf(":%s %s %s %s", serverName, errNoMotdCode,
		nick, ":MOTD file is missing")
	fmt.Println("replyNickandUserCommand:", message)
	client.conn.Write([]byte(message + crlf))
}

func replyWhoCommand(client *Client, roomName string) {
	message := ""
	room := getRoomFromName(roomName)
	isFirst := true
	temp := ""
	for clientInRoom := range room.clients {
		if isFirst {
			isFirst = false
			temp = "H@"
		} else {
			temp = "H"
		}
		message = fmt.Sprintf(":%s %s %s #%s %s %s %s %s %s :0 %s", serverName,
			rplWhoReplyCode, client.nickname, roomName,
			client.username, client.hostname,
			serverName, clientInRoom.nickname, temp, client.realname)
		fmt.Println("replyWhoCommand:", message)
		client.conn.Write([]byte(message + crlf))
	}
	message = fmt.Sprintf(":%s %s %s #%s :End of WHO list", serverName, rplEndOfWhoCode,
		client.nickname, roomName)
	fmt.Println("replyWhoCommand:", message)
	client.conn.Write([]byte(message + crlf))
}

func getRoomFromName(name string) *Room {
	for room := range rooms {
		if room.name == name {
			return room
		}
	}
	return nil
}

func getClientFromName(name string) *Client {
	for client := range clients {
		if client.nickname == name {
			return client
		}
	}
	return nil
}

func sendPingCommand(client *Client) {
	for {
		time.Sleep(1000 * time.Millisecond)
		fmt.Println("PING :" + serverName)
		client.conn.Write([]byte("PING :" + serverName))
		/*readerObj := bufio.NewReader(client.conn)
		message, _ := readerObj.ReadString('\n')
		fmt.Println("sendPingCommand:" + string(message))*/
	}
}

func handleClient(client *Client, commandChan chan *Command) {

	readerObj := bufio.NewReader(client.conn)
	//go sendPingCommand(client)
	for {
		message, _ := readerObj.ReadString('\n')
		if len(message) == 0 {
			continue
		}
		message = strings.TrimRight(message, crlf)
		if strings.Compare(strings.Split(message, " ")[0], "QUIT") == 0 {
			mutex.Lock()
			delete(clients, client)
			mutex.Unlock()
			break
		}
		fmt.Println("Message:", string(message))
		command := &Command{name: string(message), client: client}
		commandChan <- command
	}

}

func parseCommand(command *Command) {

	tokens := strings.Split(command.name, " ")

	if len(tokens) >= 2 {
		commandName := tokens[0]
		parameters := tokens[1:]
		handleFunc, ok := commandMap[commandName]
		if !ok {
			handleInvalidCommand(command)
		} else {
			handleFunc(parameters, command)
		}
	} else {
		handleInvalidCommand(command)
	}
}

func main() {

	commandChan := make(chan *Command)
	connectionChan := make(chan net.Conn)
	args := os.Args[1:]
	port := ":6667"
	if len(args) == 1 {
		port = ":" + args[0]
	}

	ln, _ := net.Listen("tcp", port)
	defer ln.Close()
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				continue
			}
			fmt.Println("accepted new connection")
			connectionChan <- conn
		}
	}()
	//server.run()
	for {
		select {
		case command := <-commandChan:
			go parseCommand(command)
		case conn := <-connectionChan:
			client := &Client{conn: conn, rooms: make(map[*Room]bool)}
			mutex.Lock()
			clients[client] = true
			mutex.Unlock()
			go handleClient(client, commandChan)
		}
	}
}
