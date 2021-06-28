- Continue working on more commands
    - ACPs:
        - Inspired by S3, etc.
        - LIST/ADD/MOD/RM
        - Algorithm (in that order)
            * If there's an explicit deny ACP, access is denied
            * If there's an explicit allow ACP, access is allowed
            * If there's no ACP, access is denied
    - CONFGET/CONFSET (only admins can CONFSET)
        * auth token expiry
        * windows names (https://stackoverflow.com/questions/1976007/what-characters-are-forbidden-in-windows-and-linux-directory-names, case insensitive)
- Fix the in-code TODOs
- Most tests could use a refactoring. Also need to be beefed up to handle all cases (regular user, single user, unauth, ACPs, etc.). Should also test for error scenarios, such as file not found.
- Ruby tests shouldn't test things with the local disk. Should just use the protocol itself
- Verify integrity of the databases when reading from them
- Allow for a custom config path (instead of .fly)
- Should be TLS by default (accept server cert like when connecting over SSH)
- Basic CLI client
    - fly cp SOURCE DEST
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
- Do we need a concept of guest user? (default anonymous)
  => could just be defined in the spec (servers need to always have a user named "guest" that has access to nothing by default)
- Put vfs & auth helpers into a separate package?
- Allow you to pass a single file instead of a dir? (for quickly sharing a file)
- Extra commands
    - SYNC
    - COPY progress report
