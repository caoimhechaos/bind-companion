Bind Companion
==============

The Bind Companion is a control process for running bind in container environments. The main purpose of the companion is to build a bind configuration file and to signal bind to reload zone configurations as they get changed.

The most reasonable use of this container is probably with the git sidecar container; as new domain zone files are pulled from git, bind is being reloaded.

The bind configuration is built from a text protocol buffer configuration file.

Volumes
-------

The Bind Companion expects the following volumes to exist:

 * **/etc/bind/git/masterzones** is the directory with the master zone files which bind is serving. Zone files are expected to have the name *domain*.db, e.g. example.org.db
 * **/etc/bind/slavezones** is the cache of slave zone files pulled from other name servers. It does not need to be on permanent storage, but since these files are supposed to help in an outage of the master domain server, it would surely help if they were.
 * **/config** contains the bind configuration protocol buffer, *bind.config*

The Master Zone directory
-------------------------

Master zones are expected to be stored under **/etc/bind/git/masterzones**. Each file should have the filename *domainname*.db, e.g. example.org.db.

A Makefile is expected to be placed in this directory which will be invoked for generating domain configurations from templates. *make* is invoked inside the master zone directory whenever a change is detected.

Since *make* is invoked as user *named*, this directory must be writable as user *named*.

Once *make* finishes, the *bind* process is signalled to reload its configuration, making any changes live.

The Bind Configuration Protocol Buffer
--------------------------------------

The format of the bind configuration protocol buffer is very simple. It essentially consists of 2 sections:

 * The domain specific configurations, and
 * the transfer permissions.

Transfer permissions are just a list of IPs which shall be allowed to request a full zone transfer (usually the slaves for master zones hosted on this server). They are stored under the key *allow_transfer*.

Domain specific configurations are stored in *domain* subsections. They have the following properties:

 * *name* specifies the domain name, without any trailing dots.
 * *type* is either *MASTER* or *SLAVE*, depending on whether the zone is hosted locally or not.
 * *master* is a text specification of the master server to pull domain configurations from if the zone is defined as type *SLAVE*.

An example configuration could look like this:

> domain {
>    name: "example.org"
>    type: MASTER
> }
>
> domain {
>    name: "example.com"
>    type: MASTER
> }
>
> domain {
>    name: "example.net"
>    type: SLAVE
>    master: "127.0.0.1"
> }
>
> allow_transfer: "127.0.0.1"
> allow_transfer: "127.0.0.2"
> allow_transfer: "127.0.1.1"

TODO
----

The following tasks are known problems which should be addressed:

 * Add Prometheus monitoring to companion job (with potential statistics
   extracted from bind process)
 * Add health check handler.
