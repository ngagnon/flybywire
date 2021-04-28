- Shouldn't use map[string]respValue where respValue has an interface{}, instead
  make respValue an interface, and have specific resp values implement its Write method
- Refactor the tests to only have 2 servers (one already configured, another empty)
- We should accept simple strings and nulls when reading command arguments
- To get the server out of single-user mode, we should use AUTH instead of ADDUSER, would make things less confusing
- Verify integrity of the databases when reading from them

- Each command should have a unit test to make sure it calls checkAuth, and to make sure it returns -DENIED when checkAuth returned false (use bytes.Buffer?)
- Handle invalid inputs in the protocol
- Allow for a custom config path (instead of .fly)
- Should allow you to pass a single file instead of a dir (for quickly sharing a file)
