{
  inputs.nixpkgs.url = "nixpkgs/nixpkgs-unstable";
  inputs.flake-utils.url = "github:numtide/flake-utils";

  outputs = { self, nixpkgs, flake-utils }:
    let
      # to work with older version of flakes
      lastModifiedDate = self.lastModifiedDate or self.lastModified or "19700101";

      # Generate a user-friendly version number.
      version = builtins.substring 0 8 lastModifiedDate;

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

          tags = [ "nix" "sdnotify" ];

          #vendorSha256 = pkgs.lib.fakeSha256;
          vendorSha256 = "a35aca9c155e9994766bad5c56d67db85476a30bd7b0864fd3191df72f340387";
        };
      in
      {
        runner = pkgs.buildGoModule (common // {
          pname = "runner";
          subPackages = [ "cmd/runner" "cmd/runner-mio" "cmd/runner-node" ];
          ldflags = (ldflags pkgs) ++ [
            "-X github.com/nyiyui/qrystal/runner.nodeUser=qrystal-node"
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
      };
      checks = (import ./test.nix) {
        inherit self system nixpkgsFor libFor nixosLibFor ldflags;
      };
      nixosModules = let peerOption = { lib }: with lib; with types; { name }: (submodule {
          options = (if name then {
            name = mkOption {
              type = str;
            };
          } else {}) // {
            host = mkOption {
              type = str;
              default = "";
            };
            allowedIPs = mkOption {
              type = listOf str;
            };
            canSee = mkOption {
              type = nullOr (oneOf [
                (submodule {
                  options = {
                    only = mkOption { type = listOf str; };
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
                    css = mkOption {
                      type = listOf (submodule {
                        options = {
                          comment = mkOption {
                            type = str;
                          };
                          endpoint = mkOption {
                            type = str;
                          };
                          tls = mkOption {
                            type = submodule {
                              options = {
                                certPath = mkOption {
                                  type = path;
                                };
                              };
                            };
                          };
                          networks = mkOption {
                            type = listOf str;
                          };
                          tokenPath = mkOption {
                            type = str;
                          };
                          azusa = mkOption { type = nullOr (submodule {
                            options = {
                              networks = mkOption {
                                type = attrsOf (peerOption { inherit lib; } { name = true; });
                              };
                            };
                          }); default = null; };
                        };
                      });
                    };
                  };
                };
              };
            };
            config = mkIf cfg.enable {
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
                            certPath = mkOption { type = path; };
                            keyPath = mkOption { type = path; };
                          };
                        };
                      };
                      addr = mkOption {
                        type = str;
                        default = ":39252";
                      };
                      harukaAddr = mkOption {
                        type = str;
                        default = ":39253";
                      };
                      tokens = mkOption {
                        type = listOf (submodule {
                          options = {
                            name = mkOption { type = str; };
                            hash = mkOption { type = str; };
                            networks = mkOption { type = nullOr (attrsOf str); };
                            canPull = mkOption { type = bool; default = false; };
                            canPush = mkOption {
                              type = nullOr (submodule {
                                options = {
                                  any = mkOption { type = bool; default = false; };
                                  networks = mkOption { type = attrsOf (submodule { options = {
                                    name = mkOption { type = str; };
                                    canSeeElement = mkOption { type = listOf str; };
                                  }; }); };
                                };
                              });
                              default = null;
                            };
                            canAddTokens = mkOption {
                              type = nullOr (submodule {
                                options = {
                                  canPull = mkOption { type = bool; default = false; };
                                  canPush = mkOption { type = bool; default = false; };
                                };
                              });
                              default = null;
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
                                  };
                                  listenPort = mkOption {
                                    type = port;
                                    default = 39390;
                                  };
                                  ips = mkOption {
                                    type = listOf str;
                                    default = [ "10.39.0/16" ];
                                  };
                                  peers = mkOption {
                                    type = attrsOf (peerOption { inherit lib; } { name = false; });
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
                  };
                };
              };
            };
          };
  });
}
