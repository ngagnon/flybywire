Data Types
===

String
---

+Lorem ipsum sit dolor amet<LF>

Map
---

%2<LF>
+First key<LF>
+First value<LF>
+Second key<LF>
+Second value<LF>

Array
---

*2<LF>
+First element<LF>
+Second element<LF>

Table
---

Simply a two-dimensional array.

=2,3<LF>
+Row 1, column 1<LF>
+Row 1, column 2<LF>
+Row 1, column 3<LF>
+Row 2, column 1<LF>
+Row 2, column 2<LF>
+Row 2, column 3<LF>

Error
---

-CODE The super duper error message<LF>

Blob
---

Dollar sign, followed by length of string, followed by line feed
Then the raw binary data, followed by a final line feed.

$5<LF>
hello<LF>

Boolean
---

#t<LF>
#f<LF>

Integer
---

:42<LF>

Null
---

_<LF>

Tag
---

Annotates another value.

@22<LF>
+Some other value

Protocol
===

The protocol is request-response.

The client starts by sending an array, where the first element is a string
representing the command name (case insensitive).

Then, the server will respond with some value, depending on the command.

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

SYNC
---

Usage: SYNC R/W path blockSize

Opens a file for syncing. Use R for reading, W for writing.

Returns a stream ID.

In reading mode, the client is expected to pass a blockSize.
Then, send the checksums for all the blocks it already has.
The last chunk it sends should be an +OK to signal the server
that it can start streaming blocks. Then the server will stream
blocks, and send NULL to terminate the stream.

In writing mode, the server will immediately start sending checksums
to the client, and end with a last +OK chunk. This signals the client
that it can start streaming blocks. It should send NULL as the
last chunk to indicate that all the blocks were sent.

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

Copies a file or folder (recursively).

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

Sets a chroot for a user. If path is NULL, then the user isn't chroot'ed.

Arguments:

- Username (string)
- Path (string or null)

Access Control
===

LISTACP
---

Shows a list of all the access control policies.

ADDACP
---

Usage: ADDACP name users[] paths perm

Creates a new access control policy.

The rule will specify:

- The affected users (colon-separated)
- A list of path prefixes (colon-separated)
- What to allow: each letter is either N (no access), R (read) or W (read-write). First letter is for file access, second letter is for ACPs.

MODACP
---

Usage: MODACP name users[] paths[] perm

Updates an access control policy. Same arguments as ADDACP.

REMACP
---

Usage: REMACP name

Deletes an access control policy.
