- CHROOT shouldn't accept invalid path (pass it through vfs.ResolveSingleUser)
- When apply chroot (in vfs), if the folder doesn't exist (or it's not a folder),
  should return NOTFOUND or something (except with LIST /)
- Should be TLS by default
    - Disabled with -notls option
- Reorganize folders: server, common/wire, cli
- Basic CLI client
    - when connecting, ask client if they want to accept server cert (like SSH)
    - fly cp SOURCE DEST

- Continue CLI client
    - fly to HOST
        - ls
        - rm
        - mv
        - mkdir
        - cd
        - cp SOURCE DEST (same as fly cp)
        - pwd
        - whoami
        - user list/add/remove/edit (l/a/r/e)
        - acp list/add/remove (l/a/r)
- Most tests could use a refactoring. Also need to be beefed up to handle all cases (regular user, single user, unauth, ACPs, etc.). Should also test for error scenarios, such as file not found.
- Ruby tests shouldn't test things with the local disk. Should just use the protocol itself
- Allow for a custom config path (instead of .fly)
- CONFGET/CONFSET (only admins can CONFSET)
    * auth token expiry
    * minimum password length
    * windows names (https://stackoverflow.com/questions/1976007/what-characters-are-forbidden-in-windows-and-linux-directory-names, case insensitive)
- Do we need a concept of guest user? (default anonymous)
  => could just be defined in the spec (servers need to always have a user named "guest" that has access to nothing by default)
- Allow you to pass a single file instead of a dir? (for quickly sharing a file)
- Client-cert TLS authentication
- Extra commands
    - SYNC
    - COPY progress report
    - ACP groups
- fly-on-s3?
- User groups
- Probably allow virtual folders to be created via the wire,
to "mount" shared folders under a user's home folder let's say

