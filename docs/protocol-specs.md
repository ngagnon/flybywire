Data Types
===

String
---

A `+` sign, followed by variable-length string, and terminated with a line feed (ASCII 10).

+Lorem ipsum sit dolor amet<LF>

Array
---

An asterisk (`*`), followed by the array length, a line feed, then the array elements.

*2<LF>
+First element<LF>
+Second element<LF>

Blob
---

A `$` sign, followed by the blob size (in bytes), the raw binary data, and then a line feed.

$5<LF>
hello<LF>

Boolean
---

A `#` followed by t (true) or f (false).

#t<LF>
#f<LF>

Integer
---

A colon (`:`), followed by the integer, then a line feed.

:42<LF>

Map
---

A `%` sign, followed by the number of key value pairs, a line feed, then alternating key-values.

%2<LF>
+First key<LF>
+First value<LF>
+Second key<LF>
+Second value<LF>

Table
---

A two-dimensional array.

A `=` sign, followed by the number of rows, then the number of columns, a line field, and then all the table values.

=2,3<LF>
+Row 1, column 1<LF>
+Row 1, column 2<LF>
+Row 1, column 3<LF>
+Row 2, column 1<LF>
+Row 2, column 2<LF>
+Row 2, column 3<LF>

Error
---

A `-` sign, followed by the error code, a space, the error message, then a line feed.

-CODE The super duper error message<LF>

Null
---

An underscore, followed by a line feed.

_<LF>

Tag
---

Annotates another value.

A `@`, followed by the tag, then a line feed, and another value.

@22<LF>
+Some other value

Connection Management
===

AUTH
---

Arguments: 

With password authentication

- PWD (string)
- Username (string)
- Password (string)

With token authentication

- TOK (string)
- Token (string)

Response:

+OK
-DENIED

TOKEN
---

When already authenticated, returns a token valid for 5 minutes that the client can use in a new connection (instead of entering the username and password again).

Response:

Authentication token (string)

PING
---

Pings the server (could be useful for keepalive).

Response:

PONG (string)

QUIT
---

Usage: QUIT

Terminates the connection.

Response:

OK (string)

File Management
===

File transfer and management commands.

These commands will not accept paths with . or .. segments.

LIST
---

Usage: LIST folder

Lists all the files under the given folder. Show the file name,
file size, and last modified time.

Instead of a folder, you could also pass a file name to get
its size and last modified time.

Would be nice: supports wildcards: * and **.

Returns:

A table with 4 columns:

- The file type (a string, D for dir, F for everything else)
- The file name (string)
- The file size in bytes (integer, or null for folders)
- The last modified time (in UTC, format: 2021-06-15T00:08:20.232167574Z)

STREAM
---

Arguments:

- R for reading, W for writing (string)
- Path (string)

Opens the given file for reading (R) or writing (W) 

New files are written to a temporary area, so they won't overwrite the
original until you're done writing it.

When opening a file for reading, the server will immediately start
sending chunks to the client via chunk responses:

@streamID\n
$length\n
blob\n

The "@" message type is a special stream message that wraps
another value, usually a blob (or an error).

When opening the file for writing, the client is expected to
send chunks to the server in the same format.

The client and server should only send chunks of up to 32KB
in size.

If an error occurs and the transfer must be stopped, an error
will be returned by the server:

@streamID\n
-ERR Error message\n

Response:

On success, returns a stream ID:

:22<LF>

Can also return errors:

-DENIED Access denied
-TOOMANY There are too many open file descriptors

CLOSE
---

Arguments:

- Stream ID (integer)

Closes the given stream ID.

Use this command when you wish to close a writing stream.

Returns:

OK (string)

MOVE
---

Usage: MOVE from to

Moves/renames a file or folder.

COPY
---

Usage: COPY from to

Copies a file from the 'from' path to the 'to' path

Returns a stream ID (integer). The server will send a null tagged with that
stream ID once the copy is completed.

DEL
---

Usage: DEL path

Deletes a file (or folder).

MKDIR
---

Arguments: 

- Path (string)

Creates a new folder.

Returns:

OK (string)

TOUCH
---

Usage: TOUCH path

Sets the last modified time of a file to now.

Response:

+OK

User Administration
===

WHOAMI
---

Usage: WHOAMI

Returns who is the currently authenticated user.

Response:

String when currently logged-in:

+john<LF>

Null when not logged-in:

_<LF>

LISTUSER
---

Usage: LISTUSER

Lists all the users.

Response:

List of usernames (array of strings)

SHOWUSER
---

Shows extra information about the user (whether he's admin, what's the chroot)

Arguments: 

- Username (string)

Response:

%3<LF>
+username<LF>
+john<LF>
+chroot<LF>
_<LF>
+admin<LF>
1<LF>

ADDUSER
---

Creates a new user.

Arguments: 

- Username (string)
- Password (string)

RMUSER
---

Deletes a user.

Arguments: 

- Username (string)

SETPWD
---

Resets a user's password.

Arguments: 

- Username (string)
- Password (string)

SETADM
---

Sets the administrator bit on the given user (has all permissions).

Arguments:

- Username (string)
- Administrator? (boolean)

CHROOT
---

Sets a chroot for a user. If path is an empty string, then the user isn't chroot'ed.

The path should be an absolute virtual path, not a physical path. If the Fly servers' root directory is /home/fly, then use a chroot like /bob, not /home/fly/bob

The folder will be created if it does not already exist.

Arguments:

- Username (string)
- Path (string)

Access Control
===

Access control in Fly is very similar to cloud services such as S3.

You create access control policies in the form of:

- Allow "bob" to read from "/home/bob"
- Deny "alice" to write in "/common"

The policies apply to any file or folder that's a descendant of the
specified path. In other words, in the first example, bob would be allowed
to read from any file or folder that starts with /home/bob, including
/home/bob.

LISTACP
---

Shows a list of all the access control policies.

Returns a table with the following fields:

- Policy name (string)
- ALLOW or DENY (string)
- R or W (string)
- Users (array of strings)
- Paths (array of strings)

PUTACP
---

Usage: PUTACP name ALLOW R/W users... paths...
Usage: PUTACP name DENY R/W users... paths...

Creates or modifies an access control policy.

Arguments:

- Rule name (string)
- ALLOW or DENY (string)
- R (read) or W (write) (string)
- Usernames (list of strings)
- Paths (list of strings)

RMACP
---

Usage: RMACP name

Deletes an access control policy.

Arguments:

- Name (string)
