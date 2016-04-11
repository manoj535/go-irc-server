package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

const (
	RPL_WELCOME_CODE    = "001"
	RPL_YOURHOST_CODE   = "002"
	RPL_CREATED_CODE    = "003"
	RPL_MYINFO_CODE     = "004"
	ERR_NOMOTD_CODE     = "422"
	RPL_WHOREPLY_CODE   = "352"
	RPL_ENDOFWHO_CODE   = "315"
	RPL_NAMREPLY_CODE   = "353"
	RPL_ENDOFNAMES_CODE = "366"
	USER_MODES          = "aio"
	CHANNEL_MODES       = "beIikntPpTl"
	JOIN_COMMAND        = "JOIN"
	MESSAGE_COMMAND     = "MSG"
	PART_COMMAND        = "PART"
	USER_COMMAND        = "USER"
	NICK_COMMNAND       = "NICK"
	WHO_COMMAND         = "WHO"
	PRIVMSG_COMMAND     = "PRIVMSG"
	SERVER_NAME         = "irc.example.com"
)

//type handleCommand func([]string)

var clients = make(map[*Client]bool)
var rooms = make(map[*Room]bool)

/*const (
	command_map = map[string]handleCommand{
		"JOIN":    handleJoinCommand,
		"MSG":     handleMessageCommand,
		"PART":    handlePartCommand,
		"USER":    handleUserCommand,
		"NICK":    handleNickCommand,
		"WHO":     handleWhoCommand,
		"PRIVMSG": handlePrivateMessageCommand,
	}
)*/

type Command struct {
	name   string
	client *Client
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
	command_chan    chan *Command
	connection_chan chan net.Conn
	name            string
}*/

func (command *Command) handleJoinCommand(parameters []string) {

	if len(parameters) == 0 {
		return
	}

	room_name := strings.TrimRight(parameters[0], "\r\n")
	if strings.ContainsAny(room_name, "#") {
		room_name = room_name[1:]
	}
	room := getRoomFromName(room_name)
	if room == nil {
		room = &Room{name: room_name, clients: make(map[*Client]bool)}
	}
	room.clients[command.client] = true
	rooms[room] = true
	replyJoinCommand(command.client, room)
}

func (command *Command) handleMessageCommand(parameters []string) {

	if len(parameters) == 0 {
		return
	}

	room_name := parameters[0]
	message := parameters[1]
	room := getRoomFromName(room_name)
	_, present := room.clients[command.client]
	if present {
		for client, _ := range room.clients {
			client.conn.Write([]byte(message + "\n"))
		}
	} else {
		command.client.conn.Write([]byte("Not part of this channel\n"))
	}
}

func (command *Command) handlePartCommand(parameters []string) {
	if len(parameters) == 0 {
		return
	}

	room_name := strings.TrimRight(parameters[0][1:], "\r\n")
	room := getRoomFromName(room_name)
	if room != nil {
		message := fmt.Sprintf(":%s!%s@%s PART #%s :%s", command.client.nickname,
			command.client.username, command.client.hostname, room_name,
			command.client.nickname)
		for room_client, _ := range room.clients {
			fmt.Println(message)
			room_client.conn.Write([]byte(message + "\r\n"))
		}
		delete(room.clients, command.client)
	}
}

func (command *Command) handleUserCommand(parameters []string) {

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

func (command *Command) handleNickCommand(parameters []string) {

	if len(parameters) != 1 {
		command.client.conn.Write([]byte("Invalid syntax\n"))
		return
	}

	nickname := strings.TrimRight(parameters[0], "\r\n")
	command.client.nickname = nickname
	replyNickAndUserCommand(command.client)
	fmt.Println("handleNickCommand")
}

func (command *Command) handleWhoCommand(parameters []string) {

	if len(parameters) != 1 {
		return
	}

	room_name := strings.TrimRight(parameters[0][1:], "\r\n")
	replyWhoCommand(command.client, room_name)
}

func (command *Command) handlePrivateMessageCommand(parameters []string) {

	if len(parameters) < 1 {
		return
	}

	room_name := strings.TrimRight(parameters[0][1:], "\r\n")
	message := strings.TrimRight(parameters[1][1:], "\r\n")
	for i := 2; i < len(parameters); i++ {
		message = message + " " + parameters[i]
	}
	final_message := ""
	room := getRoomFromName(room_name)
	_, present := room.clients[command.client]
	if present {
		for client, _ := range room.clients {
			if strings.Compare(command.client.nickname, client.nickname) != 0 {
				final_message = fmt.Sprintf(":%s!%s@%s PRIVMSG #%s :%s",
					command.client.nickname, command.client.username, client.hostname,
					room_name, message)
				fmt.Println(final_message)
				client.conn.Write([]byte(final_message + "\r\n"))
			}
		}
	} else {
		command.client.conn.Write([]byte("Not part of this channel\n"))
	}

}

func (command *Command) handleInvalidCommand() {
	command.client.conn.Write([]byte("Invalid commmand\n"))
}

func replyJoinCommand(client *Client, room *Room) {

	message := ""
	//client.nickname = "manoj"
	for room_client, _ := range room.clients {
		message = fmt.Sprintf(":%s!%s@%s %s #%s", client.nickname, client.username,
			client.hostname, "JOIN", room.name)
		fmt.Println(message)
		room_client.conn.Write([]byte(message + "\r\n"))
	}

	//message = fmt.Sprintf("%s %s #%s :%s", ":127.0.0.1", "332", channel_name, "test topic")
	//client.conn.Write([]byte(message + "\r\n"))
	//message = fmt.Sprintf("%s %s %s = #%s :@%s", ":127.0.0.1", "353", client.nickname, channel_name, client.nickname)
	message = fmt.Sprintf(":%s %s %s = #%s :@", SERVER_NAME, RPL_NAMREPLY_CODE,
		client.nickname, room.name)

	for room_client, _ := range room.clients {
		message = message + room_client.nickname + " "
	}
	message = strings.TrimRight(message, " ")
	fmt.Println(message)
	client.conn.Write([]byte(message + "\r\n"))
	message = fmt.Sprintf(":%s %s %s #%s :%s", SERVER_NAME, RPL_ENDOFNAMES_CODE,
		client.nickname, room.name, "End of NAMES list")
	fmt.Println(message)
	client.conn.Write([]byte(message + "\r\n"))

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
	message := fmt.Sprintf(":%s %s %s %s", SERVER_NAME, RPL_WELCOME_CODE,
		nick, ":Welcome to the Internet Relay Network ")
	fmt.Println("replyNickandUserCommand:", message)
	client.conn.Write([]byte(message + "\r\n"))
	// send RPL_YOURHOST
	message = fmt.Sprintf(":%s %s %s %s", SERVER_NAME, RPL_YOURHOST_CODE,
		nick, ":Your host is irc.example.com")
	fmt.Println("replyNickandUserCommand:", message)
	client.conn.Write([]byte(message + "\r\n"))
	// send RPL_CREATED
	message = fmt.Sprintf(":%s %s %s %s", SERVER_NAME, RPL_CREATED_CODE,
		nick, ":This server was created at")
	fmt.Println("replyNickandUserCommand:", message)
	client.conn.Write([]byte(message + "\r\n"))
	// send RPL_MYINFO
	message = fmt.Sprintf(":%s %s %s %s %s %s %s", SERVER_NAME, RPL_MYINFO_CODE,
		nick, "localhost", "1.0", USER_MODES, CHANNEL_MODES)
	fmt.Println("replyNickandUserCommand:", message)
	client.conn.Write([]byte(message + "\r\n"))
	// send ERR_NOMOTD
	message = fmt.Sprintf(":%s %s %s %s", SERVER_NAME, ERR_NOMOTD_CODE,
		nick, ":MOTD file is missing")
	fmt.Println("replyNickandUserCommand:", message)
	client.conn.Write([]byte(message + "\r\n"))
}

func replyWhoCommand(client *Client, room_name string) {
	message := ""
	room := getRoomFromName(room_name)
	isFirst := true
	temp := ""
	for client_in_room, _ := range room.clients {
		if isFirst {
			isFirst = false
			temp = "H@"
		} else {
			temp = "H"
		}
		message = fmt.Sprintf(":%s %s %s #%s %s %s %s %s %s :0 %s", SERVER_NAME,
			RPL_WHOREPLY_CODE, client.nickname, room_name, client.username, client.hostname,
			SERVER_NAME, client_in_room.nickname, temp, client.realname)
		fmt.Println("replyWhoCommand:", message)
		client.conn.Write([]byte(message + "\r\n"))
	}
	message = fmt.Sprintf(":%s %s %s #%s :End of WHO list", SERVER_NAME, RPL_ENDOFWHO_CODE,
		client.nickname, room_name)
	fmt.Println("replyWhoCommand:", message)
	client.conn.Write([]byte(message + "\r\n"))
}

func getRoomFromName(name string) *Room {
	for room, _ := range rooms {
		room.name = strings.TrimRight(room.name, "\r\n")
		if room.name == name {
			return room
		}
	}
	return nil
}

func handleClient(client *Client, command_chan chan *Command) {

	reader_obj := bufio.NewReader(client.conn)
	for {
		message, _ := reader_obj.ReadString('\n')
		if len(message) == 0 {
			continue
		}
		message = strings.TrimRight(message, "\r\n")
		if strings.Compare(strings.Split(message, " ")[0], "QUIT") == 0 {
			delete(clients, client)
			break
		}
		fmt.Println("Message:", string(message))
		command := &Command{name: string(message), client: client}
		command_chan <- command
	}

}

func parseCommand(command *Command) {

	tokens := strings.Split(command.name, " ")

	if len(tokens) >= 2 {
		command_name := tokens[0]
		parameters := tokens[1:]
		/*handleFunc, ok := command_map[command_name]
		if !ok {
			command.handleInvalidCommand()
		} else {
			command.handleFunc(parameters)
		}*/
		switch strings.ToUpper(command_name) {
		case JOIN_COMMAND:
			command.handleJoinCommand(parameters)
		case MESSAGE_COMMAND:
			command.handleMessageCommand(parameters)
		case PART_COMMAND:
			command.handlePartCommand(parameters)
		case USER_COMMAND:
			command.handleUserCommand(parameters)
		case NICK_COMMNAND:
			command.handleNickCommand(parameters)
		case WHO_COMMAND:
			command.handleWhoCommand(parameters)
		case PRIVMSG_COMMAND:
			command.handlePrivateMessageCommand(parameters)
		default:
			command.handleInvalidCommand()
		}
	} else {
		command.handleInvalidCommand()
	}
}

/*func (server *Server) run() {
	for {
		select {
		case command := <-server.command_chan:
			parseCommand(command)
		case conn := <-server.connection_chan:
			client := &Client{conn: conn, channels: make(map[*Channel]bool)}
			server.clients[client] = true
			go server.handleClient(client)
		}
	}
}*/

func main() {

	// listen
	//server := &Server{}
	command_chan := make(chan *Command)
	connection_chan := make(chan net.Conn)
	args := os.Args[1:]
	if len(args) != 1 {
		fmt.Println("Invalid args")
		return
	}
	port := ":" + args[0]
	ln, _ := net.Listen("tcp", port)
	defer ln.Close()
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				continue
			}
			fmt.Println("accepted new connection")
			connection_chan <- conn
		}
	}()
	//server.run()
	for {
		select {
		case command := <-command_chan:
			parseCommand(command)
		case conn := <-connection_chan:
			client := &Client{conn: conn, rooms: make(map[*Room]bool)}
			clients[client] = true
			go handleClient(client, command_chan)
		}
	}
}
