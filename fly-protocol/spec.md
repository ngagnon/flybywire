Data Types
===

String
---

+Lorem ipsum sit dolor amet<LF>

Error
---

-CODE The super duper error message<LF>

Blob
---

Dollar sign, followed by length of string, followed by line feed
Then the raw binary data, followed by a final line feed.

$5<LF>
hello<LF>

Null
---

_<LF>

Connection Management
===

AUTH
---

Usage: AUTH PWD username password

Authenticate with username & password.

Usage: AUTH TOK token

Authenticate with a token.

Response:

+OK
-DENIED

TOKEN
---

Usage: TOKEN

When already authenticated, returns a token valid for 5 minutes that the client can use in a new connection (instead of entering the username and password again).

Response:

Blob with the token in it

PING
---

Usage: PING 

Pings the server (could be useful for keepalive).

Response:

+PONG<LF>

QUIT
---

Usage: QUIT

Terminated the connection.

Response:

+OK

File Management
===

LIST
---

Usage: LIST folder

Lists all the files under the given folder. Show the file name,
file size, and last modified time.

Supports wildcards: * and **.

OPEN
---

Usage: OPEN path R/W

Opens the given file for reading (R) or writing (W) 

File descriptors are automatically closed after 1 minute of inactivity.

New files are written to a temporary area, so they won't overwrite the
original until you're done writing it.

Response:

On success, returns a file descriptor (integer):

:22<LF>

Can also return errors:

-DENIED Access denied
-TOOMANY There are too many open file descriptors

SEND
---

Usage: SEND id data

Writes a chunk of data to the given file descriptor

Set data to NULL to indicate the end of the transfer.

Response:

+OK
-CLOSED This file descriptor was closed

RECV
---

Usage: RECV id max

Reads up to `max` bytes of data from the given file descriptor.

Response:

On success, returns an array with the file descriptor as first element, and a blob as second element:

*2<LF>
:22<LF>
$5<LF>
Hello<LF>

When there are no more bytes to read:

-EOF

Other responses:

-CLOSED This file descriptor was closed

MOVE
---

Usage: MOVE from to

Moves a file or folder.

COPY
---

Usage: COPY from to

Copies a file or folder (recursively).

DEL
---

Usage: DEL path

Deletes a file (or folder).

MKDIR
---

Usage: MKDIR path

Creates a new folder.

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

SHOWUSER
---

Usage: SHOWUSER name

Shows extra information about the user (whether he's admin, what's the chroot)

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

Usage: ADDUSER name

Creates a new user.

RMUSER
---

Usage: RMUSER name

Deletes a user.

SETPWD
---

Usage: SETPWD user password

Resets a user's password.

SETADM
---

Usage: SETADM user

Make the given user an administrator (has all permissions).

NOTADM
---

Usage: NOTADM user

Make the given user a regular user.

CHROOT
---

Usage: CHROOT user path

Sets a chroot for a user. If path is NULL, then the user isn't chroot'ed.

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
