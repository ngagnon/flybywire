Access Control
===

When Fly server runs for the first time, it launches in single-user mode. In this mode, authentication and authorization are disabled, and all clients are granted full access to the server.

As soon as the first user is created, the server switches to multi-user mode. In this mode, users must authenticate before they can run commands. Access control policies are also enforced.

When a user attempts to read or write from a file, Fly gathers all the access control policies that apply to that file. It then applies the following algorithm:

- If there's at least one policy that denies access, then access is denied. (explicit deny)
- If there's at least one policy that allows access, and no denies, then access is granted.
- If there are no policies that apply to this path, then access is denied (implicit deny)


