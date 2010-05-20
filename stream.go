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
