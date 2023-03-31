{
  inputs.nixpkgs.url = "nixpkgs/nixpkgs-unstable";
  inputs.flake-utils.url = "github:numtide/flake-utils";

  outputs = { self, nixpkgs, flake-utils }:
    let
      # to work with older version of flakes
      lastModifiedDate = self.lastModifiedDate or self.lastModified or "19700101";

      # Generate a user-friendly version number.
      version = (builtins.substring 0 8 lastModifiedDate) + "-" + (if (self ? rev) then self.rev else "dirty");

      # System types to support.
      supportedSystems = [ "x86_64-linux" "aarch64-linux" ];

      # Helper function to generate an attrset '{ x86_64-linux = f "x86_64-linux"; ... }'.
      forAllSystems = nixpkgs.lib.genAttrs supportedSystems;

      # Nixpkgs instantiated for supported system 
      nixpkgsFor = forAllSystems (system: import nixpkgs { inherit system; });
      libFor = forAllSystems (system: import (nixpkgs + "/lib"));
      nixosLibFor = forAllSystems (system: import (nixpkgs + "/nixos/lib"));
    in flake-utils.lib.eachSystem supportedSystems (system: let 
      pkgs = import nixpkgs {
        inherit system;
      };
      lib = import (nixpkgs + "/lib") {
        inherit system;
      };
      nixosLib = import (nixpkgs + "/nixos/lib") {
        inherit system;
      };
      ldflags = pkgs: [
        "-X github.com/nyiyui/qrystal/mio.CommandBash=${pkgs.bash}/bin/bash"
        "-X github.com/nyiyui/qrystal/mio.CommandWg=${pkgs.wireguard-tools}/bin/wg"
        "-X github.com/nyiyui/qrystal/mio.CommandWgQuick=${pkgs.wireguard-tools}/bin/wg-quick"
        "-X github.com/nyiyui/qrystal/node.CommandIp=${pkgs.iproute2}/bin/ip"
        "-X github.com/nyiyui/qrystal/node.CommandIptables=${pkgs.iptables}/bin/iptables"
      ];
    in rec {
      devShells = let pkgs = nixpkgsFor.${system}; in { default = pkgs.mkShell {
          buildInputs = with pkgs; [
            bash
            go_1_19
            git
            protobuf
            protoc-gen-go
            protoc-gen-go-grpc
          ];
      }; };
      packages = let
        pkgs = nixpkgsFor.${system};
        lib = libFor.${system};
        common = {
          inherit version;
          src = ./.;

          ldflags = ldflags pkgs;
          buildFlags = "-race";

          tags = [ "nix" "sdnotify" ];

          #vendorSha256 = pkgs.lib.fakeSha256;
          vendorSha256 = "16025d3c73da1c3a7f34699d8360869717256a566ea8e1198f1a3983d2294ff3";
        };
      in
      {
        runner = pkgs.buildGoModule (common // {
          pname = "runner";
          subPackages = [ "cmd/runner" "cmd/runner-mio" "cmd/runner-node" "cmd/runner-hokuto" ];
          ldflags = (ldflags pkgs) ++ [
            "-X github.com/nyiyui/qrystal/runner.NodeUser=qrystal-node"
          ];
          postInstall = ''
            mkdir $out/lib
            cp $src/mio/dev-add.sh $out/lib
            cp $src/mio/dev-remove.sh $out/lib
          '';
        });
        cs = pkgs.buildGoModule (common // {
          pname = "cs";
          subPackages = [ "cmd/cs" ];
        });
        etc = pkgs.buildGoModule (common // {
          name = "etc";
          # NOTE: specifying subPackages makes buildGoModule not test other packages :(
        });
        sd-notify-test = pkgs.buildGoModule (common // {
          pname = "sd-notify-test";
          subPackages = [ "cmd/sd-notify-test" ];
        });
        dns-test = pkgs.buildGoModule (common // {
          pname = "dns-test";
          subPackages = [ "cmd/dns-test" ];
        });
      };
      checks = (import ./test.nix) {
        inherit self system nixpkgsFor libFor nixosLibFor ldflags;
      };
      nixosModules = let peerOption = { lib }: with lib; with types; { name }: (submodule {
          options = (if name then {
            name = mkOption {
              type = str;
              description = "name of the peer";
            };
          } else {}) // {
            host = mkOption {
              type = str;
              default = "";
              description = "Endpoint= in wg-quick(8) config";
            };
            allowedIPs = mkOption {
              type = nullOr (listOf str);
              description = "AllowedIPs= in wg-quick(8) config. If null, an available one is automagically allocated.";
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
                (enum [ "any" ]) # TODO: any option is not yet supported in cs config
              ]);
              default = null;
            };
          };
        }); in
        {
          node = { config, lib, pkgs, ... }: with lib; with types; let
          in let
            cfg = config.qrystal.services.node;
            mkConfigFile = cfg: builtins.toFile "node-config.json" (builtins.toJSON cfg.config);
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
                            description = "Enable and configure dnsmasq to use Hokuto DNS server";
                          };
                          addr = mkOption {
                            type = str;
                            default = "127.0.0.39";
                            description = "Hokuto bind address (no port). Leave blank to disable";
                          };
                          parent = mkOption {
                            type = str;
                            default = ".qrystal.internal";
                            description = "All domains inside networks will be of the format <peer>.<network>.<parent>";
                          };
                          useInConfig = mkOption {
                            type = bool;
                            default = true;
                            description = "Whether to use Hokuto for DNS= settings in WireGuard.";
                          };
                        };
                      };
                      default = {};
                    };
                    css = mkOption {
                      type = listOf (submodule {
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
                                  type = attrsOf (peerOption { inherit lib; } { name = true; });
                                };
                              };
                            });
                            default = null;
                            description = "Push peer to net before pulling.";
                          };
                        };
                      });
                    };
                  };
                };
              };
            };
            config = mkIf cfg.enable {
              services.dnsmasq = mkIf (cfg.config.hokuto.configureDnsmasq && cfg.config.hokuto.addr != "") {
                enable = true;
                resolveLocalQueries = true;
                servers = [ "8.8.8.8" "8.8.4.4" "/${cfg.config.hokuto.parent}/127.0.0.39" ];
                extraConfig = ''
                  conf-file=${pkgs.dnsmasq}/share/dnsmasq/trust-anchors.conf
                  dnssec
                  listen-address=::1,127.0.0.53
                  local=/${cfg.config.hokuto.parent}/
                  interface=lo
                  bind-interfaces # hokuto binds to 127.0.0.39
                '';
              };
              users.groups.qrystal-node = {};
              users.users.qrystal-node = {
                isSystemUser = true;
                description = "Qrystal Node";
                group = "qrystal-node";
              };
              systemd.services.qrystal-node = let pkg = packages.runner; in {
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
              mkConfigFile = cfg: builtins.toFile "cs-config.json" (builtins.toJSON cfg.config);
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
                      tokens = mkOption { type = listOf (submodule {
                        options = {
                          name = mkOption {
                            type = str;
                            description = "Friendly name of token.";
                          };
                          hash = mkOption {
                            type = str;
                            description = "Hash of the token (use qrystal-gen-keys).";
                          };
                          networks = mkOption {
                            type = nullOr (attrsOf str);
                            description = "Nets this token can pull from and the corresponding peer names.";
                          };
                          canPull = mkOption {
                            type = bool;
                            default = false;
                            description = "Allow token to pull peers (see networks)";
                          };
                          canPush = mkOption {
                            type = nullOr (submodule {
                              options = {
                                any = mkOption {
                                  type = bool;
                                  default = false;
                                  description = "Can push to any peer in any net.";
                                };
                                networks = mkOption { default=null;type = nullOr (attrsOf (submodule { options = {
                                  name = mkOption {
                                    type = str;
                                    description = "Peer name.";
                                  };
                                  canSeeElement = mkOption {
                                    type = listOf str;
                                    description = "Pushed peers' canSee must be an element of canSeeElement.";
                                  };
                                }; })); };
                              };
                            });
                            default = null;
                            description = "Allow token to push peers.";
                          };
                          canAdminTokens = mkOption {
                            type = nullOr (submodule {
                              options = {
                                canPull = mkOption { type = bool; default = false; description = "Tokens added by this token can arbitrarily pull."; };
                                canPush = mkOption { type = bool; default = false; description = "Tokens added by this token can arbitrarily push."; };
                              };
                            });
                            default = null;
                            description = "Allow token to add more tokens, or remove any tokens.";
                          };
                        };
                      }); };
                      central = mkOption {
                        type = submodule {
                          options = {
                            networks = mkOption {
                              type = attrsOf (submodule {
                                options = {
                                  keepalive = mkOption {
                                    type = nullOr str;
                                    default = null;
                                    description = "PersistentKeepalive= in wg-quick(8) config";
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
                                    type = attrsOf (peerOption { inherit lib; } { name = false; });
                                    description = "All peers in the net.  Note that more can be added later.";
                                    default = {};
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
                users.groups.qrystal-cs = {};
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
  });
}
