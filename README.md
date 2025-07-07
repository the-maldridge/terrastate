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

Terrastate supports multiple authentication mechanisms as defined by
the [authware](https://github.com/the-maldridge/authware) library.
Select which ones to enable by exporting `AUTHWARE_BASIC_MECHS` as a
colon seperated list of mechanisms in the order the mechanisms should
be attempted.  Its also possible to specify an authentication prefix
for projects by setting `TS_AUTH_PREFIX`. For example, if you have a
large number of projects that you wish to gate access to, its possible
to create all the groups with a common prefix:

    terrastate-staging
    terrastate-production
    terrastate-corp

When `TS_AUTH_PREFIX` is then set to terrastate-, these groups grant
access to the following hierarchies respectively:

    /state/staging/*
    /state/production/*
    /state/corp/*

If you don't want to segment your state in this way, just create a
single group that will guard access (or select an existing one) and
leave the variable empty. When unset group names will be matched
directly.
