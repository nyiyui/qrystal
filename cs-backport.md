---
title: CS Backport
---

CS Backport is intended to be a backup mechanism for CS where the tokens and
CC are stored on disk (e.g. `/etc/qrystal/cs-backport.yml`), and to provide
a single file where scripts can get a summary of the CS instanace and Nodes.
Currently, backport loading doesn't work.

## Format

```yaml
cc: <CC> # same format as used in node-config.yml and cs-config.yml
```
