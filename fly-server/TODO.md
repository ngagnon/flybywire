- Continue working on more commands
    - GETOPT/SETOPT 
        * clarify what's permanent and what's not (or maybe use different commands?)
        * chunk size negotiation
        * auth token expiry
        * windows names (https://stackoverflow.com/questions/1976007/what-characters-are-forbidden-in-windows-and-linux-directory-names, case insensitive)
    - CHROOT
    - LISTACP
    - ADDACP
    - MODACP
    - RMACP
- Could really improve the rspec tests with dynamic generation, or refactoring,
  or both. Also need to be beefed up to handle all cases (regular user, single user, unauth, ACPs, etc.). Should also test for error scenarios, such as file not found.
- Ruby tests shouldn't test things with the local disk. Should just use the protocol itself
- Verify integrity of the databases when reading from them
- Allow for a custom config path (instead of .fly)
- Allow you to pass a single file instead of a dir? (for quickly sharing a file)
- Do we need a concept of guest user? (default anonymous)
  => could just be defined in the spec (servers need to always have a user named "guest" that has access to nothing by default)
- Should be TLS by default (accept server cert like when connecting over SSH)
- Put vfs & auth helpers into a separate package?
- Basic CLI client
- Extra commands
    - COPY (would require sending progress updates over stream)
    - SYNC
