package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

const (
	RPL_WELCOME_CODE    string = "001"
	RPL_YOURHOST_CODE   string = "002"
	RPL_CREATED_CODE    string = "003"
	RPL_MYINFO_CODE     string = "004"
	ERR_NOMOTD_CODE     string = "422"
	RPL_WHOREPLY_CODE   string = "352"
	RPL_ENDOFWHO_CODE   string = "315"
	RPL_NAMREPLY_CODE   string = "353"
	RPL_ENDOFNAMES_CODE string = "366"
	USER_MODES          string = "aio"
	CHANNEL_MODES       string = "beIikntPpTl"
)

type ClientMap map[*Client]bool

type ChannelMap map[*Channel]bool

type Command struct {
	name   string
	client *Client
	server *Server
}

type Channel struct {
	name    string
	clients ClientMap
}

type Client struct {
	conn     net.Conn
	channels ChannelMap
	nickname string
	username string
	realname string
	hostname string
}

type Server struct {
	clients         ClientMap
	channels        ChannelMap
	command_chan    chan *Command
	connection_chan chan net.Conn
	name            string
}

//var server *Server

func (server *Server) handleClient(client *Client) {

	reader_obj := bufio.NewReader(client.conn)
	for {
		message, _ := reader_obj.ReadString('\n')
		if len(message) == 0 {
			continue
		}
		message = strings.TrimRight(message, "\r\n")
		if strings.Compare(strings.Split(message, " ")[0], "QUIT") == 0 {
			delete(server.clients, client)
			break
		}
		fmt.Println("Message:", string(message))
		command := &Command{name: string(message), client: client, server: server}
		server.command_chan <- command
	}

}

func (command *Command) handleJoinCommand(parameters []string) {
	if len(parameters) != 0 {
		channel_name := strings.TrimRight(parameters[0], "\r\n")
		if strings.ContainsAny(channel_name, "#") {
			channel_name = channel_name[1:]
		}
		channel := getChannelFromName(command.server, channel_name)
		if channel == nil {
			channel = &Channel{name: channel_name, clients: make(ClientMap)}
		}
		channel.clients[command.client] = true
		command.server.channels[channel] = true
		replyJoinCommand(command.client, command.server, channel_name)
		//server.clients[command.client].channels[channel] = true
		//server.channels = append(server.channels, Channel{name: channel_name})
		//fmt.Println(server.channels)
	}
}

func (command *Command) handleMessageCommand(parameters []string) {
	if len(parameters) != 0 {
		channel_name := parameters[0]
		message := parameters[1]
		channel := getChannelFromName(command.server, channel_name)
		_, present := channel.clients[command.client]
		if present {
			for client, _ := range channel.clients {
				client.conn.Write([]byte(message + "\n"))
			}
		} else {
			command.client.conn.Write([]byte("Not part of this channel\n"))
		}
	}
}

func (command *Command) handlePartCommand(parameters []string) {
	if len(parameters) != 0 {
		channel_name := strings.TrimRight(parameters[0][1:], "\r\n")
		channel := getChannelFromName(command.server, channel_name)
		if channel != nil {
			message := fmt.Sprintf(":%s!%s@%s PART #%s :%s", command.client.nickname,
				command.client.username, command.client.hostname, channel_name,
				command.client.nickname)
			for channel_client, _ := range channel.clients {
				fmt.Println(message)
				channel_client.conn.Write([]byte(message + "\r\n"))
			}
			delete(channel.clients, command.client)
		}
	}
}

func (command *Command) handleUserCommand(parameters []string) {
	if len(parameters) == 4 {
		fmt.Println("handleUserCommand")
		username := parameters[0]
		hostname := parameters[2]
		realname := parameters[3][1:]
		command.client.username = username
		command.client.realname = realname
		command.client.hostname = hostname
		//replyNickAndUserCommand(command.client)
	} else {
		command.client.conn.Write([]byte("Invalid syntax\n"))
	}
}

func (command *Command) handleNickCommand(parameters []string) {
	if len(parameters) == 1 {

		//nickname := parameters[0]
		nickname := strings.TrimRight(parameters[0], "\r\n")
		command.client.nickname = nickname
		replyNickAndUserCommand(command.client, command.server)
		fmt.Println("handleNickCommand")
	} else {
		command.client.conn.Write([]byte("Invalid syntax\n"))
	}
}

func (command *Command) handleWhoCommand(parameters []string) {
	if len(parameters) == 1 {
		channel_name := strings.TrimRight(parameters[0][1:], "\r\n")
		replyWhoCommand(command.client, command.server, channel_name)
	}
}

func (command *Command) handlePrivateMessageCommand(parameters []string) {

	if len(parameters) > 1 {
		channel_name := strings.TrimRight(parameters[0][1:], "\r\n")
		message := strings.TrimRight(parameters[1][1:], "\r\n")
		for i := 2; i < len(parameters); i++ {
			message = message + " " + parameters[i]
		}
		final_message := ""
		channel := getChannelFromName(command.server, channel_name)
		_, present := channel.clients[command.client]
		if present {
			for client, _ := range channel.clients {
				if strings.Compare(command.client.nickname, client.nickname) != 0 {
					final_message = fmt.Sprintf(":%s!%s@%s PRIVMSG #%s :%s",
						command.client.nickname, command.client.username, client.hostname,
						channel_name, message)
					fmt.Println(final_message)
					client.conn.Write([]byte(final_message + "\r\n"))
				}
			}
		} else {
			command.client.conn.Write([]byte("Not part of this channel\n"))
		}
	}

}

func (command *Command) handleInvalidCommand() {
	command.client.conn.Write([]byte("Invalid commmand\n"))
}

func replyJoinCommand(client *Client, server *Server, channel_name string) {

	message := ""
	channel := getChannelFromName(server, channel_name)
	//client.nickname = "manoj"
	for channel_client, _ := range channel.clients {
		message = fmt.Sprintf(":%s!%s@%s %s #%s", client.nickname, client.username,
			client.hostname, "JOIN", channel_name)
		fmt.Println(message)
		channel_client.conn.Write([]byte(message + "\r\n"))
	}

	//message = fmt.Sprintf("%s %s #%s :%s", ":127.0.0.1", "332", channel_name, "test topic")
	//client.conn.Write([]byte(message + "\r\n"))
	//message = fmt.Sprintf("%s %s %s = #%s :@%s", ":127.0.0.1", "353", client.nickname, channel_name, client.nickname)
	message = fmt.Sprintf(":%s %s %s = #%s :@", server.name, RPL_NAMREPLY_CODE,
		client.nickname, channel_name)

	for channel_client, _ := range channel.clients {
		message = message + channel_client.nickname + " "
	}
	message = strings.TrimRight(message, " ")
	fmt.Println(message)
	client.conn.Write([]byte(message + "\r\n"))
	message = fmt.Sprintf(":%s %s %s #%s :%s", server.name, RPL_ENDOFNAMES_CODE,
		client.nickname, channel_name, "End of NAMES list")
	fmt.Println(message)
	client.conn.Write([]byte(message + "\r\n"))

}

func replyNickAndUserCommand(client *Client, server *Server) {

	nick := ""
	if client.nickname == "" {
		nick = "*"
	} else {
		nick = client.nickname
	}
	//nick = "manoj"
	// send RPL_WELCOME
	message := fmt.Sprintf(":%s %s %s %s", server.name, RPL_WELCOME_CODE,
		nick, ":Welcome to the Internet Relay Network ")
	fmt.Println("replyNickandUserCommand:", message)
	client.conn.Write([]byte(message + "\r\n"))
	// send RPL_YOURHOST
	message = fmt.Sprintf(":%s %s %s %s", server.name, RPL_YOURHOST_CODE,
		nick, ":Your host is irc.example.com")
	fmt.Println("replyNickandUserCommand:", message)
	client.conn.Write([]byte(message + "\r\n"))
	// send RPL_CREATED
	message = fmt.Sprintf(":%s %s %s %s", server.name, RPL_CREATED_CODE,
		nick, ":This server was created at")
	fmt.Println("replyNickandUserCommand:", message)
	client.conn.Write([]byte(message + "\r\n"))
	// send RPL_MYINFO
	message = fmt.Sprintf(":%s %s %s %s %s %s %s", server.name, RPL_MYINFO_CODE,
		nick, "localhost", "1.0", USER_MODES, CHANNEL_MODES)
	fmt.Println("replyNickandUserCommand:", message)
	client.conn.Write([]byte(message + "\r\n"))
	// send ERR_NOMOTD
	message = fmt.Sprintf(":%s %s %s %s", server.name, ERR_NOMOTD_CODE,
		nick, ":MOTD file is missing")
	fmt.Println("replyNickandUserCommand:", message)
	client.conn.Write([]byte(message + "\r\n"))
}

func replyWhoCommand(client *Client, server *Server, channel_name string) {
	message := ""
	channel := getChannelFromName(server, channel_name)
	isFirst := true
	temp := ""
	for client_in_channel, _ := range channel.clients {
		if isFirst {
			isFirst = false
			temp = "H@"
		} else {
			temp = "H"
		}
		message = fmt.Sprintf(":%s %s %s #%s %s %s %s %s %s :0 %s", server.name,
			RPL_WHOREPLY_CODE, client.nickname, channel_name, client.username, client.hostname,
			server.name, client_in_channel.nickname, temp, client.realname)
		fmt.Println("replyWhoCommand:", message)
		client.conn.Write([]byte(message + "\r\n"))
	}
	message = fmt.Sprintf(":%s %s %s #%s :End of WHO list", server.name, RPL_ENDOFWHO_CODE,
		client.nickname, channel_name)
	fmt.Println("replyWhoCommand:", message)
	client.conn.Write([]byte(message + "\r\n"))
}

func getChannelFromName(server *Server, name string) *Channel {
	for channel, _ := range server.channels {
		channel.name = strings.TrimRight(channel.name, "\r\n")
		if channel.name == name {
			return channel
		}
	}
	return nil
}

func parseCommand(command *Command) {

	tokens := strings.Split(command.name, " ")

	if len(tokens) >= 2 {
		command_name := tokens[0]
		parameters := tokens[1:]
		switch strings.ToUpper(command_name) {
		case "JOIN":
			command.handleJoinCommand(parameters)
		case "MSG":
			command.handleMessageCommand(parameters)
		case "PART":
			command.handlePartCommand(parameters)
		case "USER":
			command.handleUserCommand(parameters)
		case "NICK":
			command.handleNickCommand(parameters)
		case "WHO":
			command.handleWhoCommand(parameters)
		case "PRIVMSG":
			command.handlePrivateMessageCommand(parameters)
		default:
			command.handleInvalidCommand()
		}
	} else {
		command.handleInvalidCommand()
	}
}

func (server *Server) run() {
	for {
		select {
		case command := <-server.command_chan:
			parseCommand(command)
		case conn := <-server.connection_chan:
			client := &Client{conn: conn, channels: make(ChannelMap)}
			server.clients[client] = true
			go server.handleClient(client)
		}
	}
}

func main() {

	// listen
	server := &Server{
		clients:         make(ClientMap),
		channels:        make(ChannelMap),
		command_chan:    make(chan *Command),
		connection_chan: make(chan net.Conn),
		name:            "irc.example.com",
	}
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
			server.connection_chan <- conn
		}
	}()
	server.run()
}
