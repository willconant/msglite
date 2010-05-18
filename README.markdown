Msglite
=======

Lightweight Messaging Daemon for Painless IPC


Overview
---------------

Msglite is a very simple, very fast messaging server designed to make it easier
for processes to communicate. Usually msglite listens on a Unix Domain socket,
but it can also listen on a TCP socket.

Clients can perform the following actions:

-   Send a Message

    This is a non-blocking operation. If no client is awaiting the message, it
    will be queued.
    
-   Receive a Message

    Clients receive messages by telling msglite they are ready for a message
    and then waiting for the next message from msglite.
    
    A client may specify multiple addresses on which it is ready. The first
    message to arrive on any of those addresses is sent to the client, and no
    new messages will be sent to the client until it tells msglite it is
    ready again.
    
    When a client indicates readiness to msglite, it must specify a timeout.
    If no message is received on any of the specified addresses before that
    timeout, a special timeout message is sent to the client.
    
    Multipe clients may indicate readiness on the same address, in which case,
    they form a line. Msglite will deliver messages to the first client in
    line. That client will then be removed from the line. If it indicates
    readiness on that address again, it will be placed at the end of the line.

-   Make a Query
    
    A client may query an address and then block awaiting a response. This
    process is simply a special case of sending a message and then indicating
    readiness on a randomly generated reply address.

The actual content of messages is totally opaque to msglite. It is up to you to
decide their encoding and meaning. I find that I'm pretty happy with JSON as
a wire format.

Msglite is written in the Go programming language. 

Getting Started
---------------

Follow these instructions for installing the Go programming language:

<http://golang.org/doc/install.html>

Then download and build msglite like this:

    $ git clone git://github.com/willconant/msglite.git
    $ cd msglite
    $ ./build.pl

Start msglite like this:

    $ ./msglite


Client Libraries
----------------

Currently, the only supported client library is for Perl. You can grab it here:

<http://github.com/willconant/msglite-perl>

The msglite protocol allows clients to be extremely simple. Check it out in the
sections below.


Listening on a Different Unix Socket
------------------------------------

    $ ./msglite -network unix -laddr /path/to/msglite.socket


Listening on a TCP Socket
-------------------------

    $ ./msglite -network tcp -laddr 127.0.0.1:8888

Listening on a TCP socket makes it easy to experiment with the msglite protocol
using telnet.


Adjusting Log Output
--------------------

    $ ./msglite -loglevel [minimal|info|debug]

The default setting is `info`.


Msglite Protocol
----------------

### Example

The msglite protocol is easy for both computers and humans to read and compose.
Take a look at this example:

    C: > 5 1 someAddress
    C: hello
    C: < 1 someAddress
    S: > 5 1 someAddress
    S: hello

In this example, the client sends a message with the body of "hello" to the
address "someAddress". Presumably, no other client is awaiting messages on
that address, so the message is queued.

The client then indicates readiness to receive a message on that same address,
and the server sends the queued message to the client.

### Commands

Conversations with msglite take the form of a series of so-called commands.
Each command consists of a single line terminated by `"\r\n"`. Command lines
start with a sigil indicating the type of command and containing 0 or more
arguments separated by whitespace.

In the case of message and query commands, the command line is followed by a
body of bytes indicated by the first parameter of the command, followed by
`"\r\n"`.

### Client Commands

Clients may send the following commands to the msglite server:

-   Message Commands

    `> bodyLength timeoutSeconds toAddress [replyAddress]`
    
    If bodyLength is greater than 0, the command is followed by bodyLength
    bytes followed by `"\r\n"`.
    
    The msglite server won't send any response to a message command.

-   Ready Commands

    `< timeoutSeconds onAddress1 [onAddress2..onAddress8]`

    Clients may specify up to 8 addresses to receive messages on.
    
    If a message is already queued on any of the specified addresses, the
    server will immediately respond with a message command corresponding to
    that message. The server will only send one message per ready command,
    even if there is more than one message available on the specified
    addresses. Once a message has been sent, the client is no longer
    considered ready and must send another ready command to receive further
    messages.
    
    If no messages are ready on any of the specified addresses, the client's
    readiness will be queued. When a message becomes available on any of
    the specified addresses, the server will send a message command.
    
    If no messages become available within the specified number of timeout
    seconds, the server will send a timeout command. Once a timeout command
    has been sent, the client is no longer considered ready and must send
    another ready command to receive further messages.
    
    If two or more clients indicate readiness on the same address at the
    same time, they will receive messages in the order they became ready.
    
-   Query Commands

    `? bodyLength timeoutSeconds toAddress`
    
    If bodyLength is greater than 0, the command is followed by bodyLength
    bytes followed by `"\r\n"`.
    
    Query commands act as a combination of a message and a ready command.
    
    Msglite will send your message to the indicated address appending a
    single-use reply address the recipient can use to respond to the query.
    
    If a response becomes available before the indicated number of timeout
    seconds, msglite will send a message command with that response.
    
    If no response becomes available within the indicated time, msglite will
    send a timeout command.

-   Quit Commands

    `.`
    
    The quit command will cause msglite to close the connection.

### Server Commands

Msglite may send the following commands to clients:

-   Message Commands

    Message commands from msglite to clients are formatted in exactly the same
    way describted above.

-   Timeout Commands

    `*`
    
    Timeout commands are sent when a ready command or a query command times
    out.

