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
              tls.certPath = builtins.toFile "testing-insecure-cert.pem" csCert;
              tls.keyPath = builtins.toFile "testing-insecure-key.pem" csKey;
              tokens = [
                { name = "node1"; hash = node1Hash; canPull = true; networks.testnet = "node1"; }
                { name = "node2"; hash = node2Hash; canPull = true; networks.testnet = "node2"; }
              ];
              central.networks.testnet = {
                keepalive = "10s";
                listenPort = 58120;
                ips = [ "10.123.0.1/16" ];
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
  all = let
    networkName = "testnet";
    base = { # TODO
      virtualisation.vlans = [ 1 ];
      environment.systemPackages = with pkgs; [ wireguard-tools ];
    };
  in let
    node = ({ token }: { pkgs, ... }: base // {
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
            inherit token;
          }
        ];
      };
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
            tls.certPath = builtins.toFile "testing-insecure-cert.pem" csCert;
            tls.keyPath = builtins.toFile "testing-insecure-key.pem" csKey;
            tokens = [
              { name = "node1"; hash = node1Hash; canPull = true; networks.${networkName} = "node1"; }
              { name = "node2"; hash = node2Hash; canPull = true; networks.${networkName} = "node2"; }
            ];
            central.networks.${networkName} = {
              keepalive = "10s";
              listenPort = 58120;
              ips = [ "10.123.0.1/16" ];
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
}
