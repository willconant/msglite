// Copyright (c) 2010 William R. Conant, WillConant.com
// Use of this source code is governed by the MIT licence:
// http://www.opensource.org/licenses/mit-license.php

package msglite

import (
	"strconv"
	"os"
	"io"
	"bufio"
	"strings"
)

type CommandStream struct {
	reader *bufio.Reader
	writer io.WriteCloser
	closed bool
}

func (stream *CommandStream) ReadCommand() ([]string, os.Error) {
	line, err := stream.reader.ReadString('\n')
	if err != nil {
		return nil, err
	}

	return strings.Fields(strings.TrimSpace(line)), nil
}

func (stream *CommandStream) ReadBody(bodyLen int) (string, os.Error) {
	bodyBuf := make([]byte, bodyLen + 2)
	_, err := io.ReadFull(stream.reader, bodyBuf)
	if err != nil {
		return "", err
	}
	
	if bodyBuf[bodyLen] != '\r' || bodyBuf[bodyLen+1] != '\n' {
		return "", os.NewError("body must be followed by \\r\\n")
	}
	
	return string(bodyBuf[0:bodyLen]), nil
}

func (stream *CommandStream) ReadMessage() (*Message, os.Error) {
	inCommand, err := stream.ReadCommand()
	
	if inCommand[0] == timeoutCommandStr {
		return nil, nil
	} else if inCommand[0] != messageCommandStr {
		return nil, os.NewError("invalid message from server")
	}
	
	params := inCommand[1:]
	
	if len(params) < 3 || len(params) > 4 {
		return nil, os.NewError("invalid message from server")
	}

	bodyLen, err := strconv.Atoi(params[0])
	if err != nil {
		return nil, os.NewError("invalid message from server")
	}
	
	timeout, err := strconv.Atoi64(params[1])
	if err != nil {
		return nil, os.NewError("invalid message from server")
	}
	
	toAddr := params[2]
	
	var replyAddr string
	if len(params) == 4 {
		replyAddr = params[3]
	}
	
	var body string
	
	if bodyLen > 0 {
		body, err = stream.ReadBody(bodyLen)
		if err != nil {
			return nil, err
		}
	}
	
	return &Message{toAddr, replyAddr, timeout, body, 0}, nil
}

func (stream *CommandStream) WriteCommand(command []string) os.Error {
	_, err := io.WriteString(stream.writer, strings.Join(command, " ") + "\r\n")
	return err
}

func (stream *CommandStream) WriteMessage(msg *Message) os.Error {
	if msg == nil {
		err := stream.WriteCommand([]string{timeoutCommandStr})
		return err
	}

	command := make([]string, 5)
	command[0] = messageCommandStr
	command[1] = strconv.Itoa(len(msg.Body))
	command[2] = strconv.Itoa64(msg.TimeoutSeconds)
	command[3] = msg.ToAddress
	
	if msg.ReplyAddress != "" {
		command[4] = msg.ReplyAddress
	} else {
		command = command[0:4]
	}
	
	err := stream.WriteCommand(command)
	if err != nil {
		return err
	}
	
	if len(msg.Body) > 0 {
		_, err = io.WriteString(stream.writer, msg.Body)
		if err != nil {
			return err
		}
		
		_, err = io.WriteString(stream.writer, "\r\n")
		if err != nil {
			return err
		}
	}
	
	return nil
}

func (stream *CommandStream) WriteQuery(body string, timeoutSeconds int64, toAddress string) os.Error {
	command := make([]string, 4)
	command[0] = queryCommandStr
	command[1] = strconv.Itoa(len(body))
	command[2] = strconv.Itoa64(timeoutSeconds)
	command[3] = toAddress
		
	err := stream.WriteCommand(command)
	if err != nil {
		return err
	}
	
	if len(body) > 0 {
		_, err = io.WriteString(stream.writer, body)
		if err != nil {
			return err
		}
		
		_, err = io.WriteString(stream.writer, "\r\n")
		if err != nil {
			return err
		}
	}
	
	return nil
}

func (stream *CommandStream) Close() os.Error {
	stream.closed = true
	return stream.writer.Close()
}

func (stream *CommandStream) WriteError(err os.Error) {
	stream.WriteCommand([]string{errorCommandStr, err.String()})
	stream.Close()
}

func (stream *CommandStream) WriteQuit() os.Error {
	err := stream.WriteCommand([]string{quitCommandStr})
	if err != nil {
		// we still close the stream
		stream.Close()
		return err
	}
	return stream.Close()
}
