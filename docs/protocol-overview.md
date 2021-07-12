Protocol Overview
===

At its core, Fly is a fairly simple request-response protocol running over TCP. Its default port number is 6767. Connections are encrypted with TLS unless specified otherwise.

It uses a serialization format that is both human readable and machine parsable, yet adds little overhead compared to conventional binary protocols.

The serialization format can encode common data types such as strings, binary blobs, booleans, integers, maps, arrays, and tables. It is heavily inspired by the Redis protocol. Refer to protocol specifications to learn more about the serialization format.

The client sends commands in the form of an array, with the command name passed as the first element.

The server then responds to the command with a data type appropriate for that particular command. Commands will often return the string OK to denote success, or an error message if the command failed.

Although commands are executed serially by the server, the client need not wait for the server to reply before sending its next command. This is sometimes known as pipelining.

The Fly protocol supports a wide range of commands:

- File operations (add, delete, update)
- ACL operations (grant, revoke, etc.)
- User management operations (add, delete, update)
- Efficient synchronization (similar to rsync)

They are described in more details in the protocol specifications.





