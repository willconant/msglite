Msglite
=======

What is Msglite?
----------------

Msglite is a lightweight messaging daemon with a built-in http proxy.
Here are a few of its strengths:

1. Msglite makes IPC really painless. It handles all of the queuing so
both senders and receivers of messages can be very simple.

2. The Msglite protocol is very easy to comprehend. If there isn't a
Msglite client for your language of choice, implementing one should
only take a few hours.

3. Msglite brings modern concurrency patterns like Scala's actors or
Go's channels to your existing software stack.


What about the built-in HTTP Proxy?
-----------------------------------

Msglite can also act as an HTTP proxy server designed to run upstream
from servers like Nginx. Msglite receives incoming HTTP requests and
forwards them as messages to your HTTP handlers, which are simply
Msglite clients.

In this capacity, Msglite serves as a replacement for things like
FastCGI.

The primary advantages of this approach are:

1. An HTTP request handler can reply to requests in an order other
than the order in which they were received. It can even forward a
request to another Msglite client which can handle the reply on its
own. This opens up all sorts of possibilities for writing real-time
"Comet" applications.

2. Your HTTP request handlers can gracefully restart without dropping
queued requests.

Wrapping your existing web-application in a Msglite HTTP handler is
surprisingly easy, and in the near future, we'll have Plack, Rack, and
WSGI wrappers for Perl, Ruby, and Python, respectively.


Getting Started
---------------

Msglite is written in the Go programming language, so you'll have to
install it first:

<http://golang.org/doc/install.html>

Then download and build msglite like this:

    $ git clone git://github.com/willconant/msglite.git
    $ cd msglite
    $ make

Start msglite like this:

    $ ./msglite


Documentation and Client Libraries
----------------------------------

Documentation and an up-to-date list of client libraries are available
on the Wiki:

<http://wiki.github.com/willconant/msglite/>


Copyright
---------

Copyright &copy; 2010 William R. Conant, <http://WillConant.com/>
