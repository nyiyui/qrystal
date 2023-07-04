args@{ self, system, nixpkgsFor, libFor, nixosLibFor, ldflags, ... }:
let
  pkgs = nixpkgsFor.${system};
  lib = nixosLibFor.${system} { inherit system; };
  node1Token =
    "qrystalct_/TTOsqg6hUeuODtIUj1z4aXDiU1ckks9T7/Eqod2mVrsgFC8eFdlS4fZXLBwggKO1MvI6oqoAWkiMZbHjLdP/w==";
  node1Hash =
    "qrystalcth_a2f29c49f4e3e520413f71ac2b42b5b66c0b9cc70bd757a543754d83e94ccfd8";
  node2Token =
    "qrystalct_jv4Abw0LouLeiq8GStjOsacArU56b77yyJ/XM0Nij/AoeSU7nlBFBFY87g05KCiuanyCdehtXZYg3MLxeFTI7Q==";
  node2Hash =
    "qrystalcth_75b2eb7d0cac7a796362115b5b0f267ee08eff7a87012fd4334082bba141c018";
  rootCert = builtins.readFile ./cert/minica.pem;
  rootKey = builtins.readFile ./cert/minica-key.pem;
  csCert = builtins.readFile ./cert/cs/cert.pem;
  csKey = builtins.readFile ./cert/cs/key.pem;

  autologin = { ... }: { services.getty.autologinUser = "root"; };
  base = { ... }: {
    imports = [ autologin ];
    virtualisation.vlans = [ 1 ];
    environment.systemPackages = with pkgs; [ wireguard-tools ];
    services.logrotate.enable = false; # clogs up the logs
  };
  networkBase = {
    keepalive = "10s";
    listenPort = 58120;
    ips = [ "10.123.0.1/16" ];
  };
  nodeToken = name: hash: networkNames: {
    inherit name;
    inherit hash;
    canPull = true;
    networks = builtins.foldl' (a: b: a // b) { }
      (map (networkName: { ${networkName} = name; }) networkNames);
  };
  adminTokenRaw =
    "qrystalct_0a3XVoDo0Q4Ni4b47tqSURZACuoqG0A79+LmfvkZQZsMco5P+OL/L6cbnPCKDe12Fj2kUkHWpHhw6eRypRgr8Q==";
  adminToken = {
    name = "admin";
    hash =
      "qrystalcth_98e2781b6a908f179e6df385b096decf5abde8ff8655dd30b5e55c7c4d81bb90";
    networks = null;
    canPull = true;
    canPush.any = true;
    canAdminTokens = {
      canPull = true;
      canPush = true;
    };
  };
  csTls = {
    certPath = builtins.toFile "testing-insecure-cert.pem" csCert;
    keyPath = builtins.toFile "testing-insecure-key.pem" csKey;
  };
in let
  csConfig = networkNames: token: {
    enable = true;
    config.cs = {
      comment = "cs";
      endpoint = "cs:39252";
      tls.certPath = builtins.toFile "testing-insecure-node-cert.pem"
        (rootCert + "\n" + csCert);
      networks = networkNames;
      tokenPath = builtins.toFile "token" token;
    };
  };
in {
  sd-notify-baseline = lib.runTest ({
    name = "sd-notify-baseline";
    hostPkgs = pkgs;
    nodes.machine = { pkgs, ... }: {
      systemd.services.sd-notify-test = {
        serviceConfig = {
          Type = "notify";
          ExecStart =
            "${pkgs.bash}/bin/bash -c '${pkgs.coreutils}/bin/echo notifying; ${pkgs.systemd}/bin/systemd-notify --ready & ${pkgs.coreutils}/bin/echo notified; while true; do sleep 1; done'";
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
          ExecStart = "${
              self.outputs.packages.${system}.sd-notify-test
            }/bin/sd-notify-test";
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

        environment.systemPackages = with pkgs;
          [ self.outputs.packages.${system}.etc ];

        qrystal.services.cs = {
          enable = true;
          config = {
            tls = csTls;
            tokens = [
              (nodeToken "node1" node1Hash [ "testnet" ])
              (nodeToken "node2" node2Hash [ "testnet" ])
              adminToken
            ];
            central.networks.testnet = networkBase // {
              peers.node1 = { allowedIPs = [ "10.123.0.1/16" ]; };
            };
          };
        };
      };
    };
    testScript = { nodes, ... }: ''
      import json

      cs.start()
      cs.wait_for_unit("qrystal-cs.service")
      cs.succeed("cs-admin -server 'cs:39253' -token '${adminTokenRaw}' -cert '${
        builtins.toFile "testing-insecure-cert.pem" csCert
      }' token-rm -token-hash '${node1Hash}'")
      q = json.dumps(dict(
        overwrite=True,
        name='node1new',
        hash='${node1Hash}',
        canPull=dict(
          testnet='node1',
        ),
      ))
      cs.succeed(f"echo '{q}' | cs-admin -server 'cs:39253' -token '${adminTokenRaw}' -cert '${
        builtins.toFile "testing-insecure-cert.pem" csCert
      }' token-add")
      # TODO test this actually works
    '';
  };
  all = let
    networkName = "testnet";
    networkName2 = "othernet";
    testDomain = "cs";
  in let
    node = { token }:
      { pkgs, ... }: {
        imports = [
          base
          self.outputs.nixosModules.${system}.node
        ];

        networking.firewall.allowedTCPPorts = [ 39251 ];
        qrystal.services.node = csConfig [ networkName networkName2 ] token;
        systemd.services.qrystal-node.wantedBy = [ ];
      };
  in lib.runTest ({
    name = "all";
    hostPkgs = pkgs;
    nodes = {
      node1 = node { token = node1Token; };
      node2 = node { token = node2Token; };
      cs = { pkgs, ... }: {
        imports = [ base self.outputs.nixosModules.${system}.cs ];

        networking.firewall.allowedTCPPorts = [ 39252 ];
        qrystal.services.cs = {
          enable = true;
          config = {
            tls = csTls;
            tokens = [
              (nodeToken "node1" node1Hash [ networkName networkName2 ])
              (nodeToken "node2" node2Hash [ networkName networkName2 ])
            ];
            central.networks.${networkName} = networkBase // {
              peers.node1 = {
                host = "node1:58120";
                allowedIPs = [ "10.123.0.1/32" ];
                canSee.only = [ "node2" ];
              };
              peers.node2 = {
                host = "node2:58120";
                allowedIPs = [ "10.123.0.2/32" ];
                canSee.only = [ "node1" ];
              };
            };
            central.networks.${networkName2} = networkBase // {
              keepalive = "10s";
              listenPort = 58121;
              ips = [ "10.45.0.1/16" ];
              peers.node1 = {
                host = "node1:58121";
                allowedIPs = [ "10.45.0.1/32" ];
                canSee.only = [ "node2" ];
              };
              peers.node2 = {
                host = "node2:58121";
                allowedIPs = [ "10.45.0.2/32" ];
                canSee.only = [ "node1" ];
              };
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
        node.succeed("host ${testDomain}", timeout=5) # test dnsmasq settings work
        node.systemctl("start qrystal-node.service")
        node.wait_for_unit("qrystal-node.service", timeout=20)
      print("all nodes started")
      # NOTE: there is a race condition where the peers' pubkeys could not be
      # set yet when pinged (so that's why we're using wait_until_*
      for i, node in enumerate(nodes):
        print(node.wait_until_succeeds("wg show"))
        print(node.wait_until_succeeds("wg show ${networkName}"))
        print(node.wait_until_succeeds("wg show ${networkName2}"))
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
      assert "node2.testnet.qrystal.internal has address 10.123.0.2" in pp(node1.succeed("host node2.testnet.qrystal.internal"))
      assert "node1.testnet.qrystal.internal has address 10.123.0.1" in pp(node2.succeed("host node1.testnet.qrystal.internal"))
      for node in nodes:
        assert pp(node.execute("host idkpeer.testnet.qrystal.internal 127.0.0.39"))[0] == 1
        assert pp(node.execute("host node1.idknet.qrystal.internal 127.0.0.39"))[0] == 1
      # TODO: test network level queries
    '';
  });
  azusa = let networkName = "testnet";
  in let
    node = ({ name, token, allowedIPs, canSee }:
      { pkgs, ... }: {
        imports = [ base self.outputs.nixosModules.${system}.node ];

        networking.firewall.allowedTCPPorts = [ 39251 ];
        qrystal.services.node = {
          enable = true;
          config.cs = {
            comment = "cs";
            endpoint = "cs:39252";
            tls.certPath = builtins.toFile "testing-insecure-node-cert.pem"
              (rootCert + "\n" + csCert);
            networks = [ networkName ];
            tokenPath = builtins.toFile "token" token;
            azusa.networks.${networkName} = {
              inherit name;
              host = "${name}:39251}";
              inherit allowedIPs;
              canSee.only = canSee;
            };
          };
        };
        systemd.services.qrystal-node.wantedBy = [ ];
      });
  in lib.runTest ({
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
      cs = { pkgs, ... }: {
        imports = [ base self.outputs.nixosModules.${system}.cs ];

        networking.firewall.allowedTCPPorts = [ 39252 ];
        qrystal.services.cs = {
          enable = true;
          config = {
            tls = csTls;
            tokens = [
              ((nodeToken "node1" node1Hash [ networkName ]) // {
                canPush.networks.${networkName} = {
                  name = "node1";
                  canSeeElement = [ "node2" ];
                };
                canPull = true;
                networks.${networkName} = "node1";
              })
              ((nodeToken "node2" node2Hash [ networkName ]) // {
                canPush.networks.${networkName} = {
                  name = "node2";
                  canSeeElement = [ "node2" ];
                };
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
  eo = let
    networkName = "testnet";
    eoPath = pkgs.writeShellScript "eo.sh" ''
      while IFS= read -r line
      do
        echo '{"endpoint":"1.2.3.4:5678"}'
      done
    '';
    node = { token }:
      { pkgs, ... }: {
        imports = [
          base
          self.outputs.nixosModules.${system}.node
          ({ ... }: { qrystal.services.node.config.endpointOverride = eoPath; })
        ];

        networking.firewall.allowedTCPPorts = [ 39251 ];
        qrystal.services.node = csConfig [ networkName ] token;
      };
  in lib.runTest ({
    name = "eo";
    hostPkgs = pkgs;
    nodes = {
      node1 = node { token = node1Token; };
      node2 = node { token = node2Token; };
      cs = { pkgs, ... }: {
        imports = [ base self.outputs.nixosModules.${system}.cs ];

        networking.firewall.allowedTCPPorts = [ 39252 ];
        qrystal.services.cs = {
          enable = true;
          config = {
            tls = csTls;
            tokens = [
              (nodeToken "node1" node1Hash [ networkName ])
              (nodeToken "node2" node2Hash [ networkName ])
            ];
            central.networks.${networkName} = networkBase // {
              peers.node1 = {
                host = "node1:58120";
                allowedIPs = [ "10.123.0.1/32" ];
                canSee.only = [ "node2" ];
              };
              peers.node2 = {
                host = "node2:58120";
                allowedIPs = [ "10.123.0.2/32" ];
                canSee.only = [ "node1" ];
              };
            };
          };
        };
      };
    };
    testScript = { nodes, ... }: ''
      import time

      nodes = [node1, node2]
      addrs = ["10.123.0.2", "10.123.0.1"]
      cs.start()
      cs.wait_for_unit("qrystal-cs.service")
      for node in nodes:
        node.start()
      for node in nodes:
        node.wait_for_unit("qrystal-node.service", timeout=20)
      for node in nodes:
        start = time.time()
        while True:
          now = time.time()
          if now-start > 60:
            raise RuntimeError("timeout")
          print(node.succeed("wg show ${networkName}"))
          endpoints = node.succeed("wg show ${networkName} endpoints")
          print('endpoints', endpoints)
          if ':' not in endpoints:
            # wait until sync is done
            time.sleep(1)
            continue
          assert "1.2.3.4:5678" in endpoints, "endpoint check"
          break
    '';
  });
  trace = let
    networkName = "testnet";
    eoPath = pkgs.writeShellScript "eo.sh" ''
      while IFS= read -r line
      do
        echo '{"endpoint":"1.2.3.4:5678"}'
      done
    '';
    tracePath = "/etc/qrystal-trace";
    node = { token }:
      { pkgs, ... }: {
        imports = [
          base
          self.outputs.nixosModules.${system}.node
          ({ ... }: { 
            environment.etc."qrystal-trace" = {
              text = "not yet";
              mode = "0666";
            };
            qrystal.services.node.config.trace = {
              outputPath = tracePath;
              waitUntilCNs = [ networkName ];
            };
            environment.systemPackages = with pkgs; [ go ];
          })
        ];

        networking.firewall.allowedTCPPorts = [ 39251 ];
        qrystal.services.node = csConfig [ networkName ] token;
      };
  in lib.runTest ({
    name = "trace";
    hostPkgs = pkgs;
    nodes = {
      node1 = node { token = node1Token; };
      cs = { pkgs, ... }: {
        imports = [ base self.outputs.nixosModules.${system}.cs ];

        networking.firewall.allowedTCPPorts = [ 39252 ];
        qrystal.services.cs = {
          enable = true;
          config = {
            tls = csTls;
            tokens = [
              (nodeToken "node1" node1Hash [ networkName ])
            ];
            central.networks.${networkName} = networkBase // {
              peers.node1 = {
                host = "node1:58120";
                allowedIPs = [ "10.123.0.1/32" ];
                canSee.only = [ "node1" ];
              };
            };
          };
        };
      };
    };
    testScript = { nodes, ... }: ''
      import time

      cs.start()
      cs.wait_for_unit("qrystal-cs.service")
      node1.start()
      node1.wait_for_unit("qrystal-node.service", timeout=20)
      start = time.time()
      node1.wait_until_succeeds("cat ${tracePath}", timeout=20)
      # verify trace is valid (e.g. if it finished writing correctly)
      node1.succeed("go tool trace -pprof=net ${tracePath}")
      # pprof type doesn't matter
    '';
  });
}
