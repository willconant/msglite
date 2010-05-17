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

Follow the instructions for installing the Go programming language here:

http://golang.org/doc/install.html

Then download and build msglite like this:

    $ git clone git://github.com/willconant/msglite.git
    $ cd msglite
    $ ./build.pl

Start msglite like this:

    $ ./msglite -network unix -laddr /path/to/msglite.socket


Client Libraries
----------------

Currently, the only supported client library is for Perl. You can grab it here:

    $ git clone git://github.com/willconant/msglite-perl.git

The whole thing is fewer than 160 lines. Writing a client library for your
prefered language should be trivial.


Listening on a TCP Socket
-------------------------

    $ ./msglite -network tcp -laddr 127.0.0.1:8888


Adjusting Log Output
--------------------

Msglite supports a `-loglevel` command line switch. The following values are
supported:

- minimal
- info
- debug

The default setting is `info`.
