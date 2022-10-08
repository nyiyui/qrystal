# Interconnections (i.e. forwarding to both Internet and peers)

- is forwarding necessary?
    - is there a not-accessible-from-Internet node?
- choose a forwarder
    - `can-forward` in `CentralPeer`?
    - choose "alphabetical" order or by some metric (e.g. (iperf3) speed tests?)
- setup forwarding
    - Internet
        - set `AllowedIPs` to Internet and use iptables
        - NOTE: put in `PostUp`/`PostDown` instead of running it ourselves (what if we crash?)
        - NOTE: make sure the rule is above the drop rule
    - peers
        - add the `AllowedIPs` of the peer to forward to
