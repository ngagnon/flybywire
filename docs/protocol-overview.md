Protocol Overview
===

The protocol is heavily inspired by the Redis serialization protocol. It is both human readable and machine parsable, yet it adds little overhead compared to a binary protocol.

@TODO: more about the raw protocol

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

The default port number is 6767.

