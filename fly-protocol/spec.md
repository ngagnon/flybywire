Data Types
===

Simple string
---

+blah blah blah<LF>

Error string
---

-blah blah blah<LF>

Bulk string
---

Dollar sign, followed by length of string, followed by line feed
Then the string, followed by a final line feed.

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

Bulk string with the token in it

PING
---

Usage: PING 

Pings the server (could be useful for keepalive).

Response:

+OK

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

UPLD
---

Usage: UPLD path

Uploads a file to the specified path. The server returns an ID,
which the client then uses to send chunks to the server asynchronously.

CHUNK
---

Usage: CHUNK id data

Sends a chunk for that particular file upload. 

Set data to NULL to indicate the end of the transfer.

Response:

+OK

DWNLD
---

Usage: DWNLD path

Downloads the file at the specified path. The server will
return the file in chunks asynchronously.

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

User Administration
===

WHOAMI
---

Usage: WHOAMI

Returns who is the currently authenticated user.

Response:

Simple string when currently logged-in:

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
#t<LF>

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
