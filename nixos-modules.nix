args@{ self, system, nixpkgsFor, libFor, nixosLibFor, ldflags, packages, ...
}: {
  nixosModules = let
    peerOption = { lib }:
      with lib;
      with types;
      { name }:
      (submodule {
        options = (if name then {
          name = mkOption {
            type = str;
            description = "name of the peer";
          };
        } else
          { }) // {
            host = mkOption {
              type = str;
              default = "";
              description = "Endpoint= in wg-quick(8) config";
            };
            allowedIPs = mkOption {
              type = nullOr (listOf str);
              description =
                "AllowedIPs= in wg-quick(8) config. If null, an available one is automagically allocated.";
              default = null;
            };
            canForward = mkOption {
              type = bool;
              description = "Peer can forward for other peers.";
              default = false;
            };
            canSee = mkOption {
              type = nullOr (submodule {
                options = {
                  only = mkOption {
                    type = listOf str;
                    description = "peer can only see these peers";
                  };
                };
              });
              default = null;
              description = "Peers inside the network that are visible to this peer. Null means all peers in the network are visible.";
            };
            allowedSRVs = mkOption {
              type = nullOr (listOf (submodule {
                options = {
                  service = mkOption { type = str; };
                  serviceAny = mkOption { type = bool; default = false; };
                  priorityMin = mkOption { type = port; default = 0; };
                  priorityMax = mkOption { type = port; default = 65535; };
                  weightMin = mkOption { type = port; default = 0; };
                  weightMax = mkOption { type = port; default = 65535; };
                };
              }));
              default = null;
              description = "List of allowed SRV records constraints for this peer.";
            };
          };
      });
  in {
    udptunnel-server = { config, lib, pkgs, ... }: with lib; with strings; with types; let
      cfg = config.qrystal.services.udptunnel-server;
      arguments = concatStringsSep " " [
        (optionalString (cfg.timeout != null) "--timeout ${cfg.timeout}")
        (optionalString cfg.syslog "--syslog")
        (optionalString cfg.verbose "--verbose")
      ];
    in {
        options.qrystal.services.udptunnel-server = {
          enable = mkEnableOption "udptunnel server";
          timeout = mkOption {
            type = nullOr int;
            default = null;
          };
          syslog = mkOption {
            type = bool;
            default = false;
          };
          verbose = mkOption {
            type = bool;
            default = false;
          };
          listen = mkOption {
            type = str;
          };
          destination = mkOption {
            type = str;
          };
        };
        config = mkIf cfg.enable {
          systemd.sockets.udptunnel-server = {
            description = "udptunnel server socket";
            documentation = [ "man:udptunnel(1)" ];
            socketConfig = {
              ListenStream = cfg.listen;
            };
            wantedBy = [ "sockets.target" ];
          };
          systemd.services.udptunnel-server = {
            description = "udptunnel server";
            documentation = [ "man:udptunnel(1)" ];
            requires = [ "udptunnel-server.socket" ];
            serviceConfig = {
              Type = "notify";
              Restart = "on-failure";
              ExecStart = "${pkgs.udptunnel}/bin/udptunnel --server ${arguments} ${cfg.destination}";
              StandardOutput = "journal";
              StandardError = "journal";
              DynamicUser = "yes";
              NoNewPrivileges = "yes";
              PrivateTmp = "yes";
              PrivateDevices = "yes";
              ProtectSystem = "strict";
              ProtectHome = "yes";
              ProtectKernelTunables = "yes";
              ProtectKernelModules = "yes";
              ProtectControlGroups = "yes";
              RestrictAddressFamilies = "AF_UNIX AF_INET AF_INET6";
              RestrictNamespaces = "yes";
              LockPersonality = "yes";
              MemoryDenyWriteExecute = "yes";
              RestrictRealtime = "yes";
              RemoveIPC = "yes";
              SystemCallArchitectures = "native";
            };
          };
        };
    };
    udptunnel-client = { config, lib, pkgs, ... }: with lib; with strings; with types; let
      cfg = config.qrystal.services.udptunnel-client;
      arguments = concatStringsSep " " [
        (optionalString (cfg.timeout != null) "--timeout ${cfg.timeout}")
        (optionalString cfg.syslog "--syslog")
        (optionalString cfg.verbose "--verbose")
      ];
    in {
        options.qrystal.services.udptunnel-client = {
          enable = mkEnableOption "udptunnel server";
          timeout = mkOption {
            type = nullOr int;
            default = null;
          };
          syslog = mkOption {
            type = bool;
            default = false;
          };
          verbose = mkOption {
            type = bool;
            default = false;
          };
          portal = mkOption {
            type = str;
            description = "The local socket where udp packets go in.";
          };
          server = mkOption {
            type = str;
            description = "The server to send the tunneled packets to.";
          };
        };
        config = mkIf cfg.enable {
          systemd.sockets.udptunnel-client = {
            description = "udptunnel server socket";
            documentation = [ "man:udptunnel(1)" ];
            socketConfig = {
              ListenStream = cfg.portal;
            };
            wantedBy = [ "sockets.target" ];
          };
          systemd.services.udptunnel-client = {
            description = "udptunnel server";
            documentation = [ "man:udptunnel(1)" ];
            requires = [ "udptunnel-client.socket" ];
            serviceConfig = {
              Type = "notify";
              Restart = "on-failure";
              RestartSec = 30;
              ExecStart = "${pkgs.udptunnel}/bin/udptunnel --client ${arguments} ${cfg.server}";
              StandardOutput = "journal";
              StandardError = "journal";
              DynamicUser = "yes";
              NoNewPrivileges = "yes";
              PrivateTmp = "yes";
              PrivateDevices = "yes";
              ProtectSystem = "strict";
              ProtectHome = "yes";
              ProtectKernelTunables = "yes";
              ProtectKernelModules = "yes";
              ProtectControlGroups = "yes";
              RestrictAddressFamilies = "AF_UNIX AF_INET AF_INET6";
              RestrictNamespaces = "yes";
              LockPersonality = "yes";
              MemoryDenyWriteExecute = "yes";
              RestrictRealtime = "yes";
              RemoveIPC = "yes";
              SystemCallArchitectures = "native";
            };
          };
        };
    };
    node = { config, lib, pkgs, ... }:
      with lib;
      with types;
      let
        cfg = config.qrystal.services.node;
        mkConfigFile = cfg:
          pkgs.writeText "node-config.json" (builtins.toJSON cfg.config);
      in {
        imports = [ self.outputs.nixosModules.${system}.udptunnel-client ];

        options.qrystal.services.node = {
          enable = mkEnableOption "Enables the Qrystal Node service";
          config = mkOption {
            type = submodule {
              options = {
                trace = mkOption {
                  type = nullOr (submodule {
                    options = {
                      outputPath = mkOption {
                        type = path;
                        description = ''
                          Output path of the trace. Note that PrivateTmp=yes is set on the systemd unit, so "/tmp/trace" will actually be "/tmp/systemd-private.../tmp/trace".'';
                      };
                      waitUntilCNs = mkOption {
                        type = listOf str;
                        description =
                          "Wait for these CNs to reify before stopping the trace.";
                      };
                    };
                  });
                  default = null;
                };
                endpointOverride = mkOption {
                  type = nullOr path;
                  description = "Path to executable for endpoint override.";
                  default = null;
                };
                udptunnel = mkOption {
                  type = submodule {
                    options = {
                      enable = mkEnableOption "udptunnel with endpoint override to proxy WireGuard connections.";
                      servers = mkOption {
                        type = attrsOf (attrsOf str);
                        description = "udptunnel per-node server host and port. Note that only one server is supported yet.";
                        example = ''{ examplenet.server0 = "udptunnel.example.org:1234"; }'';
                        default = {};
                      };
                    };
                  };
                  default = { enable = false; };
                };
                srvList = mkOption {
                  type = nullOr path;
                  description = "Path to list of SRV records.";
                  default = null;
                };
                hokuto = mkOption {
                  type = submodule {
                    options = {
                      configureDnsmasq = mkOption {
                        type = bool;
                        default = true;
                        description =
                          "Enable and configure dnsmasq to use Hokuto DNS server";
                      };
                      dnsmasqGoogleDNS = mkOption {
                        type = bool;
                        default = true;
                        description =
                          "Set 8.8.8.8 and 8.8.4.4 as DNS servers for dnsmasq. This is defaulted to true so if you forget to specify servers, your dnsmasq config isn't destroyed.";
                      };
                      addr = mkOption {
                        type = str;
                        default = "127.0.0.39";
                        description =
                          "Hokuto bind address (no port). Leave blank to disable";
                      };
                      parent = mkOption {
                        type = str;
                        default = ".qrystal.internal";
                        description =
                          "All domains inside networks will be of the format <peer>.<network>.<parent>";
                      };
                      extraParents = mkOption {
                        type = listOf (submodule {
                          options = {
                            network = mkOption {
                              type = nullOr str;
                              description =
                                "If not null, this parent domain will be for just this network.";
                              default = null;
                            };
                            domain = mkOption {
                              type = str;
                              description = "Parent domain.";
                              example = ".internal.example.org";
                              default = null;
                            };
                          };
                        });
                        default = [];
                        description = "Extra parent domains.";
                      };
                    };
                  };
                  default = { };
                };
                cs = mkOption {
                  type = submodule {
                    options = {
                      comment = mkOption {
                        type = str;
                        example = "main";
                        description = "Friendly name for CS";
                      };
                      endpoint = mkOption {
                        type = str;
                        example = "cs.qrystal.example.net:39252";
                        description = "Endpoint to CS";
                      };
                      tls = mkOption {
                        type = submodule {
                          options = {
                            certPath = mkOption {
                              type = path;
                              description = "Path to TLS certificate.";
                            };
                          };
                        };
                      };
                      networks = mkOption {
                        type = listOf str;
                        description = "Networks to pull from CS";
                      };
                      tokenPath = mkOption {
                        type = str;
                        example = "/run/secrets/qrystal-central-token-main";
                        description = "Path to file containing Central Token.";
                      };
                      azusa = mkOption {
                        type = nullOr (submodule {
                          options = {
                            networks = mkOption {
                              type = attrsOf
                                (peerOption { inherit lib; } { name = true; });
                            };
                          };
                        });
                        default = null;
                        description = "Push peer to net before pulling.";
                      };
                    };
                  };
                };
              };
            };
          };
        };
        config = let
          udptunnelPortal = "127.0.0.1:12345";
          udptunnelPythonScript = pkgs.writeText "qrystal-node-endpoint-override-inner.py" ''
            import json, sys

            servers_raw = """
            ${builtins.toJSON cfg.config.udptunnel.servers}
            """
            servers = json.loads(servers_raw)
            req = json.load(sys.stdin)

            def get_res(servers, req):
                endpoint = servers.get(req['cnn'], {}).get(req['pn'])
                if endpoint:
                    return dict(endpoint="${udptunnelPortal}")
                else:
                    return dict(endpoint=req.endpoint)

            res = get_res(servers, req)

            json.dump(res, sys.stdout)
          '';
        in mkIf cfg.enable {
          qrystal.services.node.config.endpointOverride = mkIf cfg.config.udptunnel.enable
            ((pkgs.writeShellScriptBin "qrystal-node-endpoint-override.sh" "${pkgs.python3}/bin/python3 -Wd ${udptunnelPythonScript}") + "/bin/qrystal-node-endpoint-override.sh");
          qrystal.services.udptunnel-client = mkIf cfg.config.udptunnel.enable {
            enable = true;
            portal = udptunnelPortal;
            server = head (attrValues (head (attrValues cfg.config.udptunnel.servers)));
          };
          services.dnsmasq = mkIf (cfg.config.hokuto.configureDnsmasq
            && cfg.config.hokuto.addr != "") {
              enable = true;
              resolveLocalQueries = true;
              settings = {
                server = [ "/${cfg.config.hokuto.parent}/${cfg.config.hokuto.addr}" ]
                  ++ (map (ep: "/${ep.domain}/${cfg.config.hokuto.addr}") cfg.config.hokuto.extraParents)
                  ++ (if cfg.config.hokuto.dnsmasqGoogleDNS then [
                    "8.8.8.8"
                    "8.8.4.4"
                  ] else
                    [ ]);
                conf-file = "${pkgs.dnsmasq}/share/dnsmasq/trust-anchors.conf";
                dnssec = true;
                listen-address = "::1,127.0.0.53";
                interface = "lo";
                bind-interfaces = true; # hokuto binds to cfg.config.hokuto.addr
              };
            };
          users.groups.qrystal-node = { };
          users.users.qrystal-node = {
            isSystemUser = true;
            description = "Qrystal Node";
            group = "qrystal-node";
          };
          systemd.services.qrystal-node = let pkg = packages.runner;
          in {
            wantedBy = [ "network-online.target" ];
            environment = {
              "RUNNER_MIO_PATH" = "${pkg}/bin/runner-mio";
              "RUNNER_HOKUTO_PATH" = "${pkg}/bin/runner-hokuto";
              "RUNNER_NODE_PATH" = "${pkg}/bin/runner-node";
              "RUNNER_NODE_CONFIG_PATH" = mkConfigFile cfg;
            } // (if (cfg.config.trace != null) then {
              "QRYSTAL_TRACE_OUTPUT_PATH" = cfg.config.trace.outputPath;
              "QRYSTAL_TRACE_UNTIL_CNS" =
                builtins.toJSON cfg.config.trace.waitUntilCNs;
            } else
              { });

            serviceConfig = {
              Restart = "on-failure";
              Type = "notify";
              NotifyAccess = "all";
              ExecStart = "${pkg}/bin/runner";
              StateDirectory = "qrystal-node";
              StateDirectoryMode = "0700";
              WorkingDirectory = "${pkg}/lib";
              PrivateTmp = "yes";
              ProtectHome = "yes";

              NoNewPrivileges = "yes";
              PrivateDevices = "yes";
              ProtectClock = "yes";
            };
          };
        };
      };
    cs = { config, lib, pkgs, ... }:
      with lib;
      with types;
      let
        cfg = config.qrystal.services.cs;
        mkConfigFile = cfg:
          builtins.toFile "cs-config.json" (builtins.toJSON cfg.config);
      in {
        options.qrystal.services.cs = {
          enable = mkEnableOption "Enables the Qrystal CS service";
          config = mkOption {
            type = submodule {
              options = {
                tls = mkOption {
                  type = submodule {
                    options = {
                      certPath = mkOption {
                        type = path;
                        description = "PEM-encoded TLS certificate";
                      };
                      keyPath = mkOption {
                        type = path;
                        description = "PEM-encoded TLS private key";
                      };
                    };
                  };
                };
                addr = mkOption {
                  type = str;
                  default = ":39252";
                  description = "Bind address of Node API";
                };
                ryoAddr = mkOption {
                  type = str;
                  default = ":39253";
                  description = "Bind address of Ryo (HTTP; for cs-push) API";
                };
                tokens = mkOption {
                  type = listOf (submodule {
                    options = {
                      name = mkOption {
                        type = str;
                        description = "Friendly name of token.";
                      };
                      hash = mkOption {
                        type = str;
                        description =
                          "Hash of the token (use qrystal-gen-keys).";
                      };
                      networks = mkOption {
                        type = nullOr (attrsOf str);
                        description =
                          "Nets this token can pull from and the corresponding peer names.";
                      };
                      canPull = mkOption {
                        type = bool;
                        default = false;
                        description =
                          "Allow token to pull peers (see networks)";
                      };
                      canPush = mkOption {
                        type = nullOr (submodule {
                          options = {
                            any = mkOption {
                              type = bool;
                              default = false;
                              description = "Can push to any peer in any net.";
                            };
                            networks = mkOption {
                              default = null;
                              type = nullOr (attrsOf (submodule {
                                options = {
                                  name = mkOption {
                                    type = str;
                                    description = "Peer name.";
                                  };
                                  canSeeElement = mkOption {
                                    type =
                                      (oneOf [ (listOf str) (enum [ "any" ]) ]);
                                    description =
                                      "Pushed peers' canSee must be an element of canSeeElement.";
                                  };
                                };
                              }));
                            };
                          };
                        });
                        default = null;
                        description = "Allow token to push peers.";
                      };
                      canAdminTokens = mkOption {
                        type = nullOr (submodule {
                          options = {
                            canPull = mkOption {
                              type = bool;
                              default = false;
                              description =
                                "Tokens added by this token can arbitrarily pull.";
                            };
                            canPush = mkOption {
                              type = bool;
                              default = false;
                              description =
                                "Tokens added by this token can arbitrarily push.";
                            };
                          };
                        });
                        default = null;
                        description =
                          "Allow token to add more tokens, or remove any tokens.";
                      };
                      canSRVUpdate = mkOption {
                        type = bool;
                        description = "Allow updating SRV allowanes for peers it can push to.";
                        default = false;
                      };
                      srvAllowances = mkOption {
                        type = nullOr (listOf (submodule {
                          options = {
                            service = mkOption { type = str; };
                            serviceAny = mkOption { type = bool; default = false; };
                            priorityMin = mkOption { type = port; default = 0; };
                            priorityMax = mkOption { type = port; default = 65535; };
                            weightMin = mkOption { type = port; default = 0; };
                            weightMax = mkOption { type = port; default = 65535; };
                          };
                        }));
                        default = null;
                      };
                      srvAllowancesAny = mkOption {
                        type = bool;
                        description = "Allow updating SRV allowances without restrictions by srvAllowances.";
                        default = false;
                      };
                    };
                  });
                };
                central = mkOption {
                  type = submodule {
                    options = {
                      networks = mkOption {
                        type = attrsOf (submodule {
                          options = {
                            keepalive = mkOption {
                              type = nullOr str;
                              default = null;
                              description =
                                "PersistentKeepalive= in wg-quick(8) config";
                            };
                            listenPort = mkOption {
                              type = port;
                              default = 39390;
                              description = "ListenPort= in wg-quick(8) config";
                            };
                            ips = mkOption {
                              type = listOf str;
                              default = [ "10.39.0/16" ];
                              description = "Endpoint= in wg-quick(8) config";
                            };
                            peers = mkOption {
                              type = attrsOf
                                (peerOption { inherit lib; } { name = false; });
                              description =
                                "All peers in the net.  Note that more can be added later.";
                              default = { };
                            };
                          };
                        });
                      };
                    };
                  };
                };
              };
            };
          };
        };
        config = mkIf cfg.enable {
          users.groups.qrystal-cs = { };
          users.users.qrystal-cs = {
            isSystemUser = true;
            description = "Qrystal CS";
            group = "qrystal-cs";
          };
          systemd.services.qrystal-cs = {
            wants = [ "network-online.target" ];
            wantedBy = [ "multi-user.target" ];
            serviceConfig = let pkg = packages.cs;
            in {
              User = "qrystal-cs";
              Restart = "on-failure";
              Type = "notify";
              ExecStart = "${pkg}/bin/cs -config ${mkConfigFile cfg}";
              RuntimeDirectory = "qrystal-cs";
              RuntimeDirectoryMode = "0700";
              StateDirectory = "qrystal-cs";
              StateDirectoryMode = "0700";
              LogsDirectory = "qrystal-cs";
              LogsDirectoryMode = "0700";
              PrivateTmp = "yes";
              ProtectHome = "yes";
            };
          };
        };
      };
  };
}
