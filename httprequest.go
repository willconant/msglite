// Most of this file is borrowed from src/pkg/http/request.go in the Go source distribution.

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the GOLICENSE file.

package msglite

import (
	"bytes"
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type httpRequest struct {
	method string
	url string
	protocol string
	headers map[string]string
	body []byte
}

const (
	maxLineLength  = 4096 // assumed <= bufio.defaultBufSize
	maxValueLength = 4096
	maxHeaderLines = 1024
)

type badStringError struct {
	what string
	str  string
}

func (e *badStringError) String() string {
	return fmt.Sprintf("%s %q", e.what, e.str)
}

type requestError struct {
	os.ErrorString
}

var (
	errLineTooLong = &requestError{"header line too long"}
	errHeaderTooLong = &requestError{"header too long"}
)

var validMethodRegexp = regexp.MustCompile("^(HEAD|GET|POST|PUT|DELETE)$")
var validProtocolRegexp = regexp.MustCompile("^HTTP/1\\.[01]$")

const (
	statusContinue           = "100"
	statusSwitchingProtocols = "101"
	
	statusOK                   = "200"
	statusCreated              = "201"
	statusAccepted             = "202"
	statusNonAuthoritativeInfo = "203"
	statusNoContent            = "204"
	statusResetContent         = "205"
	statusPartialContent       = "206"
	
	statusMultipleChoices   = "300"
	statusMovedPermanently  = "301"
	statusFound             = "302"
	statusSeeOther          = "303"
	statusNotModified       = "304"
	statusUseProxy          = "305"
	statusTemporaryRedirect = "307"
	
	statusBadRequest                   = "400"
	statusUnauthorized                 = "401"
	statusPaymentRequired              = "402"
	statusForbidden                    = "403"
	statusNotFound                     = "404"
	statusMethodNotAllowed             = "405"
	statusNotAcceptable                = "406"
	statusProxyAuthRequired            = "407"
	statusRequestTimeout               = "408"
	statusConflict                     = "409"
	statusGone                         = "410"
	statusLengthRequired               = "411"
	statusPreconditionFailed           = "412"
	statusRequestEntityTooLarge        = "413"
	statusRequestURITooLong            = "414"
	statusUnsupportedMediaType         = "415"
	statusRequestedRangeNotSatisfiable = "416"
	statusExpectationFailed            = "417"
	
	statusInternalServerError     = "500"
	statusNotImplemented          = "501"
	statusBadGateway              = "502"
	statusServiceUnavailable      = "503"
	statusGatewayTimeout          = "504"
	statusHTTPVersionNotSupported = "505"
)

var statusText = map[string]string{
	statusContinue:           "Continue",
	statusSwitchingProtocols: "Switching Protocols",
	
	statusOK:                   "OK",
	statusCreated:              "Created",
	statusAccepted:             "Accepted",
	statusNonAuthoritativeInfo: "Non-Authoritative Information",
	statusNoContent:            "No Content",
	statusResetContent:         "Reset Content",
	statusPartialContent:       "Partial Content",
	
	statusMultipleChoices:   "Multiple Choices",
	statusMovedPermanently:  "Moved Permanently",
	statusFound:             "Found",
	statusSeeOther:          "See Other",
	statusNotModified:       "Not Modified",
	statusUseProxy:          "Use Proxy",
	statusTemporaryRedirect: "Temporary Redirect",
	
	statusBadRequest:                   "Bad Request",
	statusUnauthorized:                 "Unauthorized",
	statusPaymentRequired:              "Payment Required",
	statusForbidden:                    "Forbidden",
	statusNotFound:                     "Not Found",
	statusMethodNotAllowed:             "Method Not Allowed",
	statusNotAcceptable:                "Not Acceptable",
	statusProxyAuthRequired:            "Proxy Authentication Required",
	statusRequestTimeout:               "Request Timeout",
	statusConflict:                     "Conflict",
	statusGone:                         "Gone",
	statusLengthRequired:               "Length Required",
	statusPreconditionFailed:           "Precondition Failed",
	statusRequestEntityTooLarge:        "Request Entity Too Large",
	statusRequestURITooLong:            "Request URI Too Long",
	statusUnsupportedMediaType:         "Unsupported Media Type",
	statusRequestedRangeNotSatisfiable: "Requested Range Not Satisfiable",
	statusExpectationFailed:            "Expectation Failed",
	
	statusInternalServerError:     "Internal Server Error",
	statusNotImplemented:          "Not Implemented",
	statusBadGateway:              "Bad Gateway",
	statusServiceUnavailable:      "Service Unavailable",
	statusGatewayTimeout:          "Gateway Timeout",
	statusHTTPVersionNotSupported: "HTTP Version Not Supported",
}

// This is lifted right out of request.go in the http package.
func readLineBytes(b *bufio.Reader) (p []byte, err os.Error) {
	if p, err = b.ReadSlice('\n'); err != nil {
		// We always know when EOF is coming.
		// If the caller asked for a line, there should be a line.
		if err == os.EOF {
			err = io.ErrUnexpectedEOF
		}
		return nil, err
	}
	if len(p) >= maxLineLength {
		return nil, errLineTooLong
	}

	// Chop off trailing white space.
	var i int
	for i = len(p); i > 0; i-- {
		if c := p[i-1]; c != ' ' && c != '\r' && c != '\t' && c != '\n' {
			break
		}
	}
	return p[0:i], nil
}

// and so is this
var colon = []byte{':'}
func readKeyValue(b *bufio.Reader) (key, value string, err os.Error) {
	line, e := readLineBytes(b)
	if e != nil {
		return "", "", e
	}
	if len(line) == 0 {
		return "", "", nil
	}

	// Scan first line for colon.
	i := bytes.Index(line, colon)
	if i < 0 {
		goto Malformed
	}

	key = string(line[0:i])
	if strings.Index(key, " ") >= 0 {
		// Key field has space - no good.
		goto Malformed
	}

	// Skip initial space before value.
	for i++; i < len(line); i++ {
		if line[i] != ' ' {
			break
		}
	}
	value = string(line[i:])

	// Look for extension lines, which must begin with space.
	for {
		c, e := b.ReadByte()
		if c != ' ' {
			if e != os.EOF {
				b.UnreadByte()
			}
			break
		}

		// Eat leading space.
		for c == ' ' {
			if c, e = b.ReadByte(); e != nil {
				if e == os.EOF {
					e = io.ErrUnexpectedEOF
				}
				return "", "", e
			}
		}
		b.UnreadByte()

		// Read the rest of the line and add to value.
		if line, e = readLineBytes(b); e != nil {
			return "", "", e
		}
		value += " " + string(line)

		if len(value) >= maxValueLength {
			return "", "", &badStringError{"value too long for key", key}
		}
	}
	return key, value, nil

Malformed:
	return "", "", &badStringError{"malformed header line", string(line)}
}

var cmap = make(map[string]string)
func canonicalHeaderKey(s string) string {
	if t, ok := cmap[s]; ok {
		return t
	}

	// canonicalize: first letter upper case
	// and upper case after each dash.
	// (Host, User-Agent, If-Modified-Since).
	// HTTP headers are ASCII only, so no Unicode issues.
	a := []byte(s)
	upper := true
	for i, v := range a {
		if upper && 'a' <= v && v <= 'z' {
			a[i] = v + 'A' - 'a'
		}
		if !upper && 'A' <= v && v <= 'Z' {
			a[i] = v + 'a' - 'A'
		}
		upper = false
		if v == '-' {
			upper = true
		}
	}
	t := string(a)
	cmap[s] = t
	return t
}

func readHttpRequest(reader io.Reader) (*httpRequest, os.Error) {	
	b := bufio.NewReader(reader)

	reqLineBytes, err := readLineBytes(b)
	if err != nil {
		return nil, err
	}
	
	var reqLineFields []string
	if reqLineFields = strings.Split(string(reqLineBytes), " ", 3); len(reqLineFields) < 3 {
		return nil, &badStringError{"malformed HTTP request", string(reqLineBytes)}
	}
	
	method, url, protocol := reqLineFields[0], reqLineFields[1], reqLineFields[2]
	
	if !validMethodRegexp.MatchString(method) {
		return nil, &badStringError{"invalid request method", method}
	}
	
	if !validProtocolRegexp.MatchString(protocol) {
		return nil, &badStringError{"invalid protocol", protocol}
	}
	
	headers := make(map[string]string)
	
	nheader := 0
	for {
		var key, value string
		if key, value, err = readKeyValue(b); err != nil {
			return nil, err
		}
		if key == "" {
			break
		}
		if nheader++; nheader >= maxHeaderLines {
			return nil, errHeaderTooLong
		}

		key = canonicalHeaderKey(key)

		// RFC 2616 says that if you send the same header key
		// multiple times, it has to be semantically equivalent
		// to concatenating the values separated by commas.
		oldvalue, present := headers[key]
		if present {
			headers[key] = oldvalue + "," + value
		} else {
			headers[key] = value
		}
	}
	
	contentLength := -1
	if clStr, ok := headers["Content-Length"]; ok {
		contentLength, err = strconv.Atoi(clStr)
		if err != nil {
			return nil, err
		}
	} else if method == "HEAD" || method == "GET" {
		contentLength = 0
	}
	
	var body []byte
	
	if contentLength == -1 {
		body, err = ioutil.ReadAll(b)
		if err != nil {
			return nil, err
		}
	} else {
		body = make([]byte, contentLength)
		if contentLength > 0 {
			io.ReadAtLeast(b, body, contentLength)
		}
	}
	
	return &httpRequest{method, url, protocol, headers, body}, nil
}
