terrastate
==========

terrastate is a simplified remote state daemon for HashiCorp
Terraform.  This daemon implements a remote state storage engine, and
a locking function.

To run the system you must select an authentication backend, specify
by the `TS_AUTH` environment variable.  Two backends are included by
default.  A local backend `file` looks in a file for usernames and
passwords in plaintext seperated by a comma, one per line:

```
user1:pass1
user2:pass2
```

The second backend uses a remote NetAuth server to provide
authentication services.  The NetAuth backend is more advanced, and
will match users to namespaces according to the rule of
`terrastate-<namespace>`.  In order to be allowed to write to the
namespace the user much posses the correct groups.
