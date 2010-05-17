package msglite

import (
	"fmt"
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

func (stream *CommandStream) ReadCommand() (string, map[string]string, os.Error) {
	line, err := stream.reader.ReadString('\n')
	if err != nil {
		return "", nil, err
	}
	
	command := line[0:len(line)-1]
	
	headers := make(map[string]string)
	for {
		line, err := stream.reader.ReadString('\n')
		if err != nil {
			return "", nil, err
		}
		
		if line == "\n" {
			break
		}
		
		sepIdx := strings.Index(line, " ")
		header := line[0:sepIdx]
		value := line[sepIdx+1:len(line)-1]

		headers[header] = value
	}
	
	return command, headers, nil
}

func (stream *CommandStream) ReadBody(bodyLen int) (string, os.Error) {
	bodyBuf := make([]byte, bodyLen + 1)
	_, err := io.ReadFull(stream.reader, bodyBuf)
	if err != nil {
		return "", err
	}
	
	if bodyBuf[bodyLen] != '\n' {
		return "", os.NewError("body must be followed by newline")
	}
	
	return string(bodyBuf[0:bodyLen]), nil
}

func (stream *CommandStream) WriteCommand(command string, headers map[string]string) os.Error {
	_, err := io.WriteString(stream.writer, command + "\n")
	if err != nil {
		return err
	}
	
	for header, value := range(headers) {
		_, err := io.WriteString(stream.writer, header + " " + value + "\n")
		if err != nil {
			return err
		}
	}
	return nil
}

func (stream *CommandStream) WriteMessage(msg Message) os.Error {
	headers := make(map[string]string)
	headers[toHeaderStr] = msg.ToAddress
	if msg.ReplyAddress != "" {
		headers[replyHeaderStr] = msg.ReplyAddress
	}
	headers[timeoutHeaderStr] = strconv.Itoa64(msg.TimeoutSeconds)
	headers[bodyLenHeaderStr] = strconv.Itoa(len(msg.Body))

	err := stream.WriteCommand(messageCommandStr, headers)
	if err != nil {
		return err
	}
	
	_, err = io.WriteString(stream.writer, "\n")
	if err != nil {
		return err
	}
	
	_, err = io.WriteString(stream.writer, msg.Body)
	if err != nil {
		return err
	}
	
	_, err = io.WriteString(stream.writer, "\n")
	if err != nil {
		return err
	}
	
	return nil
}

func (stream *CommandStream) Close() os.Error {
	stream.closed = true
	return stream.writer.Close()
}

func (stream *CommandStream) WriteError(err os.Error) {
	io.WriteString(stream.writer, fmt.Sprintf(errorCommandStr + "\n" + errorHeaderStr + " %v\n\n", err))
	stream.Close()
}

func (stream *CommandStream) WriteQuit() os.Error {
	_, err := io.WriteString(stream.writer, quitCommandStr + "\n\n")
	if err != nil {
		// we still close the stream
		stream.Close()
		return err
	}
	return stream.Close()
}
