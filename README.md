terrastate
==========

terrastate is a simplified remote state daemon for HashiCorp
Terraform.  This daemon implements a remote state storage engine, and
a locking function.

To run the system you must select an authentication backend, specify
by the `--auth_backend` flag.  Two backends are included by default.
A local backend `file` looks in a file for usernames and passwords in
plaintext seperated by a comma, one per line:

```
user1:pass1
user2:pass2
```

The second backend uses are remote NetAuth server to provide
authentication services.  Users can be required to have a specific
group.  If no group is specified, then any user can authenticate and
use the server.
