---
title: SRV records
---

[Hokuto](hokuto) supports settings SRV records per-peer. These are controlled by SRV allowances, which are in turn controlled by token-level SRV allowances.

## Getting Started

1. (Only if using azusa to push the peer you will use SRV records for.) Set token-level SRV allowances:

```nix
{
  qrystal.services.cs.config.tokens = [{
    # name, hash, etc
    canSRVUpdate = true; # required for updating SRV records of any peer through this token
    srvAllowancesAny = true; # if true, no restrictions on SRV allowances through this token
    srvAllowances = [{
      # see nixos-modules.nix for details
    }];
  }];
}
```

2. Set peer-level SRV allowances.

```nix
{
  qrystal.services.cs.config.central.networks.testnet.peers.testpeer = {
    allowedSRVs = [{
      service = "_sip"; # only SRV records with service equal to "_sip" allowed
      serviceAny = false;
      priorityMin = 0;
      priorityMax = 0;
      weightMin = 0;
      weightMax = 0;
    }];
  };
}
```

3. Provide a srv list to a node.

```nix
{
  qrystal.services.node.config.srvList = pkgs.writeText "srvlist.json" (builtins.toJSON { Networks = {
    msb = [{
      Service = "_sip";
      Protocol = "_udp";
      Priority = 0;
      Weight = 0;
      Port = 5060;
    }];
  }; });
}
```
