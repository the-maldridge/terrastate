# LDAP Authentication

LDAP is a supported authentication backend for gating access to
terraform state.  The operation of the LDAP backend functions very
similarly to the NetAuth backend with a few additional config options.

## Configuration Options

To select the LDAP Auth backend, start terrastate with the `TS_AUTH`
variable set to `ldap`.

Additionally, you must configure the following variables:

    * `TS_LDAP_URL`: A URL starting with either `ldap://` or `ldaps://`
    * `TS_LDAP_BASEDN`: The root path to search under for users
    * `TS_LDAP_GROUPATTR`: The attribute on a user that specifies
      groups.  Unless you know why you're setting this to something
      different, it should usually be set to `memberOf`.
    * `TS_LDAP_BIND_TEMPLATE`: The UID template that a user will bind
      as.  Specify as a string with `%s` where the username will go.

## Specifying Groups

Finally, you can also set `TS_AUTH_PREFIX` to a common prefix that
will be added to project names prior to matching group membership.
For example, if you have a large number of projects that you wish to
gate access to, its possible to create all the groups with a common
prefix:

  * terrastate-staging
  * terrastate-production
  * terrastate-corp

When `TS_AUTH_PREFIX` is then set to `terrastate-`, these groups grant
access to the following hierarchies respectively:

  * /state/staging/*
  * /state/production/*
  * /state/corp/*

If you don't want to segment your state in this way, just create a
single group that will guard access (or select an existing one) and
leave the variable empty.  When unset group names will be matched
directly.
