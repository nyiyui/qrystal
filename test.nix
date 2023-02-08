{ self, system, nixpkgsFor, libFor, nixosLibFor, ldflags, ... }:
let
  pkgs = nixpkgsFor.${system};
  lib = nixosLibFor.${system} { inherit system; };
  node1Token = "qrystalct_/TTOsqg6hUeuODtIUj1z4aXDiU1ckks9T7/Eqod2mVrsgFC8eFdlS4fZXLBwggKO1MvI6oqoAWkiMZbHjLdP/w==";
  node1Hash  = "qrystalcth_a2f29c49f4e3e520413f71ac2b42b5b66c0b9cc70bd757a543754d83e94ccfd8";
  node2Token = "qrystalct_jv4Abw0LouLeiq8GStjOsacArU56b77yyJ/XM0Nij/AoeSU7nlBFBFY87g05KCiuanyCdehtXZYg3MLxeFTI7Q==";
  node2Hash  = "qrystalcth_75b2eb7d0cac7a796362115b5b0f267ee08eff7a87012fd4334082bba141c018";
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

          environment.systemPackages = with pkgs; [
            host
          ];

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
      def pp(value):
          print("pp", value)
          return value
      assert "node2.testnet.qrystal.internal has address 10.123.0.2" in pp(node1.succeed("host node2.testnet.qrystal.internal 127.0.0.1"))
      assert "node1.testnet.qrystal.internal has address 10.123.0.1" in pp(node2.succeed("host node1.testnet.qrystal.internal 127.0.0.1"))
      for node in nodes:
        assert pp(node.execute("host idkpeer.testnet.qrystal.internal 127.0.0.1"))[0] == 1
        assert pp(node.execute("host node1.idknet.qrystal.internal 127.0.0.1"))[0] == 1
      # TODO: test network level queries
    '';
  });
  azusa = let
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
    name = "azusa";
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
                canPull = true;
                networks.${networkName} = "node1";
              })
              ((nodeToken "node2" node2Hash networkName) // {
                canPush.networks.${networkName} = { name = "node2"; canSeeElement = [ "node2" ]; };
                canPull = true;
                networks.${networkName} = "node2";
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
      for node in nodes:
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
