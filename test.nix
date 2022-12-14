{ self, system, nixpkgsFor, libFor, nixosLibFor, ldflags, ... }:
let
  pkgs = nixpkgsFor.${system};
  lib = nixosLibFor.${system} { inherit system; };
  node1Token = "3ztQRDsLo+iOEeU8BJp7GTiAhrpMr8rLt5HrAlDUwNEItGhcjW98lsCyKIpmCT+AtkC1vLDkfRSvWk1JQlMVlw==";
  node1Hash = "72d487c5632716c8cdf3cf440ed29e14171e27c245c715f92e2517aee605fc71";
  node2Token = "oiiOXY3dRRgmm/DjPQFJja5OftVBrZffRpxyUWk2CWabdAn9jUfeIh4vlSE35eJK0qjfb7w/2/XJya/xoduNew==";
  node2Hash = "b7ba9038ed61f27a1e4142007eaaed7a0309fed9180bace0474d2891bada9395";
  rootCert = builtins.readFile ./cert/minica.pem;
  rootKey = builtins.readFile ./cert/minica-key.pem;
  csCert = builtins.readFile ./cert/cs/cert.pem;
  csKey = builtins.readFile ./cert/cs/key.pem;
in let
  base = { # TODO
    virtualisation.vlans = [ 1 ];
    environment.systemPackages = with pkgs; [ wireguard-tools ];
  };
  networkBase = {
    keepalive = "10s";
    listenPort = 58120;
    ips = [ "10.123.0.1/16" ];
  };
  nodeToken = name: hash: networkName: {
    inherit name;
    inherit hash;
    canPull = true;
    networks.${networkName} = name;
  };
  csTls = {
    certPath = builtins.toFile "testing-insecure-cert.pem" csCert;
    keyPath = builtins.toFile "testing-insecure-key.pem" csKey;
  };
in let
  csConfig = networkName: token: {
    enable = true;
    config.css = [
      {
        comment = "cs";
        endpoint = "cs:39252";
        tls.certPath = builtins.toFile "testing-insecure-node-cert.pem" (rootCert + "\n" + csCert);
        networks = [ networkName ];
        tokenPath = builtins.toFile "token" token;
      }
    ];
  };
in
{
  sd-notify-baseline = lib.runTest ({
    name = "sd-notify-baseline";
    hostPkgs = pkgs;
    nodes.machine = { pkgs, ... }: {
      systemd.services.sd-notify-test = {
        serviceConfig = {
          Type = "notify";
          ExecStart = "${pkgs.bash}/bin/bash -c '${pkgs.coreutils}/bin/echo notifying; ${pkgs.systemd}/bin/systemd-notify --ready & ${pkgs.coreutils}/bin/echo notified; while true; do sleep 1; done'";
        };
      };
    };
    testScript = ''
      machine.start()
      machine.systemctl("start sd-notify-test.service")
      machine.wait_for_unit("sd-notify-test.service")
    '';
  });
  sd-notify = lib.runTest ({
    name = "sd-notify";
    hostPkgs = pkgs;
    nodes.machine = { pkgs, ... }: {
      systemd.services.sd-notify-test = {
        serviceConfig = {
          Type = "notify";
          ExecStart = "${self.outputs.packages.${system}.sd-notify-test}/bin/sd-notify-test";
        };
      };
    };
    testScript = ''
      machine.start()
      machine.systemctl("start sd-notify-test.service")
      machine.wait_for_unit("sd-notify-test.service")
    '';
  });
  cs = lib.runTest {
      name = "cs";
      hostPkgs = pkgs;
      nodes = {
        cs = { pkgs, ... }: {
          imports = [ self.outputs.nixosModules.${system}.cs ];

          qrystal.services.cs = {
            enable = true;
            config = {
              tls = csTls;
              tokens = [
                (nodeToken "node1" node1Hash "testnet")
                (nodeToken "node2" node2Hash "testnet")
              ];
              central.networks.testnet = networkBase // {
                peers.node1 = { allowedIPs = [ "10.123.0.1/16" ]; };
              };
            };
          };
        };
      };
      testScript = { nodes, ... }: ''
        cs.start()
        cs.wait_for_unit("qrystal-cs.service")
      '';
    };
  push = let
    networkName = "testnet";
  in let
    node = ({ token }: { pkgs, ... }: base // {
      imports = [ self.outputs.nixosModules.${system}.node ];
      qrystal.services.node = csConfig networkName token;
      systemd.services.qrystal-node.wantedBy = [];
    });
  in
  lib.runTest ({
    name = "push";
    hostPkgs = pkgs;
    nodes = {
      node1 = node { token = node1Token; }; # pushing for node1
      node2 = node { token = node2Token; };
      pusher = { pkgs, ... }: base // {
        environment.systemPackages = [ self.outputs.packages.${system}.etc ];
      };
      cs = { pkgs, ... }: base // {
        imports = [ self.outputs.nixosModules.${system}.cs ];

        networking.firewall.allowedTCPPorts = [ 39252 39253 ];
        qrystal.services.cs = {
          enable = true;
          config = {
            tls = csTls;
            tokens = [
              ((nodeToken "node2" node2Hash networkName) // {
                canPush.networks.${networkName} = { name = "node1"; canSeeElement = [ "node2" ]; };
                canAddTokens = { canPull = true; };
              })
            ];
            central.networks.${networkName} = networkBase // {
              peers.node2 = { host = "node2:58120"; allowedIPs = [ "10.123.0.2/32" ]; canSee.only = [ "node1" ]; };
            };
          };
        };
      };
    };
    testScript = { nodes, ... }: ''
      import json

      nodes = [node1, node2]
      addrs = ["10.123.0.2", "10.123.0.1"]
      cs.start()
      cs.wait_for_unit("qrystal-cs.service")
      node2.start()
      node2.wait_for_unit("qrystal-node.service")
      config = dict(
        overwrite=False,
        name="node1",
        tokenHash="${node1Hash}",
        networks=dict(
          ${networkName}=dict(
            name="node1",
            ips=["10.123.0.1/32"],
            host="node1:58120",
            canSee=["node2"],
          ),
        ),
      )
      pusher.start()
      print(pusher.succeed(f"""${self.outputs.packages.${system}.etc}/bin/cs-push -server 'cs:39253' -token '${node2Token}' -cert <(echo '${rootCert + "\n" + csCert}') -tmp-config <(echo '{json.dumps(config)}')"""))
      print("pushed for node1")
      node1.start()
      node1.wait_for_unit("qrystal-node.service")
      print("all nodes started")
      # NOTE: there is a race condition where the peers' pubkeys could not be
      # set yet when pinged (so that's why we're using wait_until_*
      for i, node in enumerate(nodes):
        print(node.wait_until_succeeds("wg show ${networkName}"))
        print(node.execute("cat /etc/wireguard/${networkName}.conf")[1])
        print(node.execute("ip route show")[1])
        for addr in addrs:
          print(node.execute(f"ip route get {addr}")[1])
      for i, node in enumerate(nodes):
        print(node.execute(f"ping -c 1 {addrs[i]}")[1])
        node.wait_until_succeeds(f"ping -c 1 {addrs[i]}")
    '';
  });
  all = let
    networkName = "testnet";
  in let
    node = ({ token }: { pkgs, ... }: base // {
      imports = [ self.outputs.nixosModules.${system}.node ];

      networking.firewall.allowedTCPPorts = [ 39251 ];
      qrystal.services.node = csConfig networkName token;
      systemd.services.qrystal-node.wantedBy = [];
    });
  in
  lib.runTest ({
    name = "all";
    hostPkgs = pkgs;
    nodes = {
      node1 = node { token = node1Token; };
      node2 = node { token = node2Token; };
      cs = { pkgs, ... }: base // {
        imports = [ self.outputs.nixosModules.${system}.cs ];

        networking.firewall.allowedTCPPorts = [ 39252 ];
        qrystal.services.cs = {
          enable = true;
          config = {
            tls = csTls;
            tokens = [
              (nodeToken "node1" node1Hash networkName)
              (nodeToken "node2" node2Hash networkName)
            ];
            central.networks.${networkName} = networkBase // {
              peers.node1 = { host = "node1:58120"; allowedIPs = [ "10.123.0.1/32" ]; canSee.only = [ "node2" ]; };
              peers.node2 = { host = "node2:58120"; allowedIPs = [ "10.123.0.2/32" ]; canSee.only = [ "node1" ]; };
            };
          };
        };
      };
    };
    testScript = { nodes, ... }: ''
      nodes = [node1, node2]
      addrs = ["10.123.0.2", "10.123.0.1"]
      cs.start()
      cs.wait_for_unit("qrystal-cs.service")
      for node in nodes:
        node.start()
        node.systemctl("start qrystal-node.service")
        node.wait_for_unit("qrystal-node.service", timeout=20)
      print("all nodes started")
      # NOTE: there is a race condition where the peers' pubkeys could not be
      # set yet when pinged (so that's why we're using wait_until_*
      for i, node in enumerate(nodes):
        print(node.wait_until_succeeds("wg show ${networkName}"))
        print(node.execute("cat /etc/wireguard/${networkName}.conf")[1])
        print(node.execute("ip route show")[1])
        for addr in addrs:
          print(node.execute(f"ip route get {addr}")[1])
      for i, node in enumerate(nodes):
        print(node.execute(f"ping -c 1 {addrs[i]}")[1])
        node.wait_until_succeeds(f"ping -c 1 {addrs[i]}")
    '';
  });
  all-push = let
    networkName = "testnet";
  in let
    node = ({ name, token, allowedIPs, canSee }: { pkgs, ... }: base // {
      imports = [ self.outputs.nixosModules.${system}.node ];

      networking.firewall.allowedTCPPorts = [ 39251 ];
      qrystal.services.node = {
        enable = true;
        config.css = [
          {
            comment = "cs";
            endpoint = "cs:39252";
            tls.certPath = builtins.toFile "testing-insecure-node-cert.pem" (rootCert + "\n" + csCert);
            networks = [ networkName ];
            tokenPath = builtins.toFile "token" token;
            azusa.networks.${networkName} = {
              inherit name;
              host = "${name}:39251}";
              inherit allowedIPs;
              canSee.only = canSee;
            };
          }
        ];
      };
      systemd.services.qrystal-node.wantedBy = [];
    });
  in
  lib.runTest ({
    name = "all-push";
    hostPkgs = pkgs;
    nodes = {
      node1 = node {
        name = "node1";
        token = node1Token;
        allowedIPs = [ "10.123.0.1/32" ];
        canSee = [ "node2" ];
      };
      node2 = node {
        name = "node2";
        token = node2Token;
        allowedIPs = [ "10.123.0.2/32" ];
        canSee = [ "node2" ];
      };
      cs = { pkgs, ... }: base // {
        imports = [ self.outputs.nixosModules.${system}.cs ];

        networking.firewall.allowedTCPPorts = [ 39252 ];
        qrystal.services.cs = {
          enable = true;
          config = {
            tls = csTls;
            tokens = [
              ((nodeToken "node1" node1Hash networkName) // {
                canPush.networks.${networkName} = { name = "node1"; canSeeElement = [ "node2" ]; };
                canAddTokens = { canPull = true; };
              })
              ((nodeToken "node2" node2Hash networkName) // {
                canPush.networks.${networkName} = { name = "node2"; canSeeElement = [ "node2" ]; };
                canAddTokens = { canPull = true; };
              })
            ];
            central.networks.${networkName} = networkBase;
          };
        };
      };
    };
    testScript = { nodes, ... }: ''
      nodes = [node1, node2]
      addrs = ["10.123.0.2", "10.123.0.1"]
      cs.start()
      cs.wait_for_unit("qrystal-cs.service")
      for node in nodes:
        node.start()
        node.systemctl("start qrystal-node.service")
        node.wait_for_unit("qrystal-node.service", timeout=20)
      print("all nodes started")
      # NOTE: there is a race condition where the peers' pubkeys could not be
      # set yet when pinged (so that's why we're using wait_until_*
      for i, node in enumerate(nodes):
        print(node.wait_until_succeeds("wg show ${networkName}"))
        print(node.execute("cat /etc/wireguard/${networkName}.conf")[1])
        print(node.execute("ip route show")[1])
        for addr in addrs:
          print(node.execute(f"ip route get {addr}")[1])
      for i, node in enumerate(nodes):
        print(node.execute(f"ping -c 1 {addrs[i]}")[1])
        node.wait_until_succeeds(f"ping -c 1 {addrs[i]}")
    '';
  });
}
