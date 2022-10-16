Centralsource provides a cenrtal place for configs for Nodes.

Nodes connect to a CS instance (a gRPC server) and the CS instance streams
changes as they are made.
CS also allows changing the config during runtime using token authentication.
The current config is saved to a backport path
(e.g. `/etc/qrystal/cs-backport.yml`) for backup and recovery purposes
(the recovery code is currently broken).
