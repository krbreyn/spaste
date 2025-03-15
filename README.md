Socket Paste

A basic command-line based implementation of a PasteBin server.

It accepts URLs to shorten via netcat/nc over TCP, such as `cat code.file| netcat server port -N`.

It will print a key to you, which, you can fetch by running `curl serverurl {key returned}`.
