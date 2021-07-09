Fly-by-wire (abbreviated Fly) is a simple file transfer protocol for the modern age. The repo includes the protocol specification, as well as reference client & server implementations.

The project's main ambition is to address some of the shortcomings of the FTP protocol. It's main selling points include:

- Firewall-friendly (only uses a single TCP port)
- Connections encrypted with TLS by default
- Commands & binary data are multiplexed over a single connection
- Uses an internal user database independent from system users
- Supports powerful access control policies inspired by S3 bucket policies
- Users and access control are managed directly via the protocol
- (WIP) Mirrors files & folders efficiently, similarly to rsync

Progress
===

The project is currently a work in progress. Use at your own risk.

Server:

- Implements 100% of the protocol

Client:

- Only supports file upload & download

Further Reading
===

- Protocol overview
- Protocol specification

Server
===

Usage: fly-server ROOTDIR

The Fly server operates within a root directory on the machine. It should
have full read/write access inside it.

Users and ACL rules are stored directly inside the root dir by default,
under a folder named .fly (invisible to clients).

The server responds to port 6767 by default. It encrypts connections
with TLS by default

Options:

-port: change the port number
-notls: disables TLS



