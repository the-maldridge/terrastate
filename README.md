terrastate
==========

terrastate is a simplified remote state daemon for HashiCorp
Terraform.  This daemon implements a remote state storage engine, and
a locking function.  You use it by configuring a `backend` block in
your Terraform configuration like this:

```hcl
terraform {
  backend "http" {
    address = "http://terrastate.example.com/state/<namespace>/<project>"
    lock_address = "http://terrastate.example.com/state/<namespace>/<project>"
    unlock_address = "http://terrastate.example.com/state/<namespace>/<project>"
  }
}
```

You will also need to configure an authentication mechanism.  There
are several currently supported.  Select an authentication backend by
exporting the environment variable `TS_AUTH` with the name of the
backend you want to enable.  Multiple backends may be separated by
colons and will be tried in the order they are listed
(`TS_AUTH='backend1:backend2'`).

## File Backend

At the very simplist is the file backend.  You should not use the file
backend, it exists to make testing much easier, or in cases where you
would like to have no authentication in front of your Terraform state
(do not do this).

The file backend uses a plain text file at a location specified by
`TS_USER_FILE`.  The file contains username and password pairs, and
looks like this:

```
user1:pass1
user2:pass2
```

The passwords are stored in plain text, and the File backend is not
group-aware, so all users have access to all projects.

## HTPassword Backend

If you want to use local authentication that goes in a file, use the
`htpasswd` backend.  This backend uses two files, defined by
`TS_HTPASSWD_FILE` and `TS_HTGROUP_FILE`.  The `htpasswd` file is, as
the name implies, formatted like an Apache2 `htpasswd` file.  You can
use any of the following algorithms with this implementation:

  * SSHA
  * MD5Crypt
  * APR1Crypt
  * SHA
  * Bcrypt
  * Plain text
  * Crypt with SHA-256 and SHA-512

This backend is group-aware, and the `htgroup` file must exist, and
uses space separated lists of users to form groups.  All groups will
be prefixed by `TS_AUTH_PREFIX` which by default is set to
`terrastate-`.  This means that your groups should contain this prefix
as well:

```
terrastate-namespace1: user1 user2
terrastate-namespace2: user1
```

## NetAuth Backend

The Netauth backend uses [NetAuth](https://netauth.org/) to provide a
remote information source for users and groups.  The backend is
configured by the system NetAuth configuration file, and respects the
value of `TS_AUTH_PREFIX` as described above.

The NetAuth backend will identify all requests by asserting the
ServiceName as `terrastate`.  This value is not configurable.

## LDAP Backend

The LDAP backend checks auth against an LDAP server.  It is described
in more detail [here](internal/web/auth/ldap/).
