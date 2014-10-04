pushed
=====

A push daemon written in Go.

Requirements
-------------

- Postgresql 9.x

Startup
-------

Create the pushed PostgreSQL user and database running `./quickinit.sh <postgres_user>`.

Systemd support
----------------

- Edit `misc/systemd/pushed.service` according to your configuration.
- Move this file to `/usr/lib/systemd/system/pushed.service`.
- Start the service on boot with:
```sh
$ systemct enable pushed.service
```
