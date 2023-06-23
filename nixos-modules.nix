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
            canSee = mkOption {
              type = nullOr (oneOf [
                (submodule {
                  options = {
                    only = mkOption {
                      type = listOf str;
                      description = "peer can only see these peers";
                    };
                  };
                })
                (enum [
                  "any"
                ]) # TODO: any option is not yet supported in cs config
              ]);
              default = null;
            };
          };
      });
  in {
    node = { config, lib, pkgs, ... }:
      with lib;
      with types;
      let
      in let
        cfg = config.qrystal.services.node;
        mkConfigFile = cfg:
          builtins.toFile "node-config.json" (builtins.toJSON cfg.config);
      in {
        options.qrystal.services.node = {
          enable = mkEnableOption "Enables the Qrystal Node service";
          config = mkOption {
            type = submodule {
              options = {
                hokuto = mkOption {
                  type = submodule {
                    options = {
                      configureDnsmasq = mkOption {
                        type = bool;
                        default = true;
                        description =
                          "Enable and configure dnsmasq to use Hokuto DNS server";
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
        config = mkIf cfg.enable {
          services.dnsmasq = mkIf (cfg.config.hokuto.configureDnsmasq
            && cfg.config.hokuto.addr != "") {
              enable = true;
              resolveLocalQueries = true;
              settings = {
                server = [ "/${cfg.config.hokuto.parent}/127.0.0.39" ];
                conf-file = "${pkgs.dnsmasq}/share/dnsmasq/trust-anchors.conf";
                dnssec = true;
                listen-address = "::1,127.0.0.53";
                local = "/${cfg.config.hokuto.parent}/";
                interface = "lo";
                bind-interfaces = true; # hokuto binds to 127.0.0.39
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
            };

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
                  description = "Bind address of Ryo (HTTP) API";
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
                                    type = listOf str;
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
            wantedBy = [ "network-online.target" ];
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
