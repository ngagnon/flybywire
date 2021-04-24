Fly-by-wire (abbreviated Fly) is a simple file transfer protocol, as well
as a reference client and server.

Its main goal is to fix some of the shortcomings of the FTP protocol

- Having to open up multiple ports in your firewall
- No way to administer the server directly via the protocol
- Server shouldn't have to manage the client's current working directory

@TODO: fly-on-s3?

Protocol
===

The protocol is based on RESP3 (Redis Serialization Protocol) to make
client libraries easy to implement.

It supports:

- File operations (add, delete, update)
- ACL operations (grant, revoke, etc.)
- User management operations (add, delete, update)
- Efficient synchronization (similar to rsync)

Files can be downloaded and uploaded using chunk encoding.

By default, the server starts out in single-user mode, where anyone who connects to the server can do whatever they want.

As soon as the first admin user is created, the server switches to multi-user mode, where everything becomes denied by default, and ACL allow rules must be
created.

Users can be chrooted to a specific directory. Users can have quotas as well.

Automatic ACL rules can be created based on the username, for instance to
give each user full access to the folder with their name.

When evaluating the ACLs on a file or folder, the least permissive ACL
is used?

@TODO: probably allow virtual folders to be created via the wire,
to "mount" shared folders under a user's home folder let's say

The default port number is 6767.

Server
===

Usage: fly-server ROOTDIR

The Fly server operates within a root directory on the machine. It should
have full read/write access inside it.

Users and ACL rules are stored directly inside the root dir by default,
under a folder named .fly (invisible to clients).



