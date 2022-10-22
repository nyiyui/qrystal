#!/usr/bin/env python3
import yaml
import json
import os
import sys

def ping(host: str):
    print('pinging', host, file=sys.stderr)
    if '"' in host:
        return False
    return 0 == os.system(f"ping -c 1 \"{host}\" > /dev/null")

res = '```\n'

with open("/etc/qrystal/cs-backport.yml") as f:
    backport = yaml.safe_load(f)

    cc = backport["cc"]
    for cnn, cn in cc['networks'].items():
        res += f"{cnn}:\n"
        for pn, peer in cn['peers'].items():
            ips = []
            for ip in peer['allowed-ips']:
                if ip.endswith('/32'): # ignor IPv6 for now
                    ips.append('○' if ping(ip[:-3]) else '×' + ip)
                else:
                    ips.append('?' + ip)
            res += f"  {pn}:\n"
            res += f"    forwarding-for: {peer['forwarding-peers']}\n"
            res += f"    allowed-ips: {ips}\n"

    res += "tokens:"
    for hash, token in backport['tokens'].items():
        token = json.loads(token)
        res += f"  {hash.split(':')[1][:6]}: {token['Name']}\n"

res += '```'
print(json.dumps(res))
