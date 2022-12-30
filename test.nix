{ self, system, nixpkgsFor, libFor, nixosLibFor, ldflags, ... }:
let
  pkgs = nixpkgsFor.${system};
  lib = nixosLibFor.${system} { inherit system; };
  node1Token = "3ztQRDsLo+iOEeU8BJp7GTiAhrpMr8rLt5HrAlDUwNEItGhcjW98lsCyKIpmCT+AtkC1vLDkfRSvWk1JQlMVlw==";
  node1Hash = "72d487c5632716c8cdf3cf440ed29e14171e27c245c715f92e2517aee605fc71";
  node2Token = "oiiOXY3dRRgmm/DjPQFJja5OftVBrZffRpxyUWk2CWabdAn9jUfeIh4vlSE35eJK0qjfb7w/2/XJya/xoduNew==";
  node2Hash = "b7ba9038ed61f27a1e4142007eaaed7a0309fed9180bace0474d2891bada9395";
  rootCert = ''
-----BEGIN CERTIFICATE-----
MIIDSzCCAjOgAwIBAgIIAl+Lb+ns3swwDQYJKoZIhvcNAQELBQAwIDEeMBwGA1UE
AxMVbWluaWNhIHJvb3QgY2EgMDI1ZjhiMCAXDTIyMTIyNjA1MTMwMVoYDzIxMjIx
MjI2MDUxMzAxWjAgMR4wHAYDVQQDExVtaW5pY2Egcm9vdCBjYSAwMjVmOGIwggEi
MA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQCoeoT2L45Dr7xWAx4TZjuj+1Kl
7z3q7zX8vdzYk4ZZYpi4JkZ5+urwu8PMBQCAC1MoX4WTKhqs2Swo4AW9oNjg2cnO
a4RYUFej+0yTV/Kdt+bxjXMwVYJDYTicX/Sx8ByQrxI55mkXW5DdDcq/BxxByGmG
XafARcJaf0nrn20lcto2KS0camGamGFGkhJdAjVHxxX7QuW/ygPkJC7VXOw5TcAN
8hQLr/arViRuJJqtCRFECoG5lkXfllQCcbKBjiP0nDYjCwviAFFglhb9Zq6JddYL
TdcqFApyAwPm33eYGM/7LtG5kCpWMMYWhmj7VIK98UOeFHnCCtYA7XrdaObxAgMB
AAGjgYYwgYMwDgYDVR0PAQH/BAQDAgKEMB0GA1UdJQQWMBQGCCsGAQUFBwMBBggr
BgEFBQcDAjASBgNVHRMBAf8ECDAGAQH/AgEAMB0GA1UdDgQWBBRC63gaY8+PMYdX
HOzZZjt4MaPxhDAfBgNVHSMEGDAWgBRC63gaY8+PMYdXHOzZZjt4MaPxhDANBgkq
hkiG9w0BAQsFAAOCAQEAIbXMM9NFOLLjH5FkQ8ZWBu8kkfDfVbPpHkITMw3YUhna
2InhfaPGlE4NzFPhjkFIgKmzPfxzlOHomVzHmk4dgf4Audl2X30tHxGBtHCGBDZD
JqHl9CaxQ9Ui4x4zR95Oi00T5PATovyNnE3Rd17AoGnADn56Ua9S+icx9TjoF1v8
NC/Om3aJ7Xnqfqo1Dfum9RkmzlGuc2jns4fo9wwwEHnG5rFBvBRUMVE27X21BJsv
zg0J2x/LrAmscMZn0Ika2SIh6g3KoBYnDU1m0hIpRgoBEFUQxqb4LxBQR5xhWZwH
QI8SZVj7mptDTcSmkRzgDpY6zGoO7VxU3fkHnBB+ug==
-----END CERTIFICATE-----
'';
  rootKey = ''
-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAqHqE9i+OQ6+8VgMeE2Y7o/tSpe896u81/L3c2JOGWWKYuCZG
efrq8LvDzAUAgAtTKF+FkyoarNksKOAFvaDY4NnJzmuEWFBXo/tMk1fynbfm8Y1z
MFWCQ2E4nF/0sfAckK8SOeZpF1uQ3Q3KvwccQchphl2nwEXCWn9J659tJXLaNikt
HGphmphhRpISXQI1R8cV+0Llv8oD5CQu1VzsOU3ADfIUC6/2q1YkbiSarQkRRAqB
uZZF35ZUAnGygY4j9Jw2IwsL4gBRYJYW/WauiXXWC03XKhQKcgMD5t93mBjP+y7R
uZAqVjDGFoZo+1SCvfFDnhR5wgrWAO163Wjm8QIDAQABAoIBACRRxTgNKG4PBFrG
cUVdVJ4VH8wFtyNeThUeGO3XX68FQkbweWDyZpNe5uakbWctCdA6R2FiQj3g01Q8
dwBaHGbcjFSjePRQ3ZPMKMXav8KgUnjgNWTGCj7cRofvZ6C0UnQeSZ+RvDX8103Q
G1TzA3Rq79S3e+JHJ466wgS5aZ4YuqquLy/9u7ojjxCoFsV4bVPfkt5exIhrHjZR
G277+IhPh65cAObp1A4OMi1wYvBMuqkcFcUcOY2KAv+/CfIgq5CXUuNQhNtJ8YOE
C5bFwdSMKQUX0PkC9IUOjm4Kbdz9pQqUDqFgtD/qP6PIrufFw0y+iV8zoPPva1Fm
tb9swAECgYEA2lACX0XGDbpDGO65yhFYOIEIXvkNbDdQRWoIO+aGmUpSqfaqfn+y
yIru1R3tsAcOpEhzUkRiTk/9D3qsJR5K/h8TFwt1PvfVC1z9BXai47s6sMhnys3b
TDdHhPKcPknzXm4rx3XGoebe+9LYr8OI0ttuSeZ15+qHASLhWXnKKzECgYEAxZAs
tq2HacSDgM0iE903/3PiQMIbVEsEyS2VGqaDRTrLP+C1MAWPrmspjW96wE631dQK
A6h7mgePsMitUDF+VVu3chtK2lO40A/2uqTkZbn5FnPhTLUn+PAqwyWtsvAA4HED
vjUDakWZHmCFFfv8xd+UQodBytG1th7z7/3QB8ECgYEAiYTq6Z7nMpCJYbRHnm0s
mHNXlZPnC6sQSpmPVERTt04lImF6ZrMEKOWzqtXuevsHEx98XW8sSc6DR3Pr6nnZ
nZhviw2xrpepQT4zOHTSCQhQ4TlsgEkKgkk0KSA2odotjudxdnTPSf9HqXPZAWb3
0nNdVvnwfcWzg1i4gYeBfZECgYAD7YzmCOczVCPlMK7nxDMz0gMClJlkgKVUtqJL
SFo9yyB1YatYjBPCPQEzfa7sGeSPzMpyLixe8J2Lv0Gq4YEIg21PSHmhg56eDGM0
bMjZuOvZ5W3qT4O+8E95V8tvTlRGIhkX9AfgWgfkUbjzqfHpoTtaY0QMm0TInS7u
a5ZyQQKBgQC5p8gV+U/kp7jNQFuGUbYL0xIFVBaKZf4mP7hcq7pPkx5ej5Udrgu7
X6lfL3K7Gjg1Qa6BM3tQ8odS1vbfuB2j+6klSthNJW5ChW79tjzkfZZTQ9JIxKcF
8nOG++nxINqj4bJG+7vsp0Sg7jwi94bBo5BNDUFqbuXr0soate8eEQ==
-----END RSA PRIVATE KEY-----
'';
  csCert = ''
-----BEGIN CERTIFICATE-----
MIIDHjCCAgagAwIBAgIIATrnPnVsh48wDQYJKoZIhvcNAQELBQAwIDEeMBwGA1UE
AxMVbWluaWNhIHJvb3QgY2EgMDI1ZjhiMB4XDTIyMTIyNjA1MTMwMVoXDTI1MDEy
NTA1MTMwMVowDTELMAkGA1UEAxMCY3MwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAw
ggEKAoIBAQCgLOhKRh6/44dflWoor5bVnqHgcb2SIMBPZHX5GjXy7V1CG8sDy12/
cnHGsuVQi9PtM/PFe1zKxNPkFiPp81RNwzGSBkfP3P0Y5t4jLpJEr7V+jbP0ag6f
QHbqrlB20E0UyT5eTC+8zzkmlmwZjSuZikJ2KEHckyNEb1b/jXsF3phGOky3qCZw
MaY5njQIFyWC/9MRa/i/Dfx+qbL02mqaXKUJhB4OExaf4xWP8U6TbluOthzuZ4Bf
MdrP2wYmUjP8xAT0VGNqxi4UDsJN8xe4+sDzBIYoXWqO0G5wPymUq0TRlR13C1dv
XbH2fsya0jF4aJbQRSMKQ+mh/EAs2YdJAgMBAAGjbzBtMA4GA1UdDwEB/wQEAwIF
oDAdBgNVHSUEFjAUBggrBgEFBQcDAQYIKwYBBQUHAwIwDAYDVR0TAQH/BAIwADAf
BgNVHSMEGDAWgBRC63gaY8+PMYdXHOzZZjt4MaPxhDANBgNVHREEBjAEggJjczAN
BgkqhkiG9w0BAQsFAAOCAQEAoxefM68W/FJ3Vy2dnoLKmiZWFuaabMatVzU37ciZ
arvKvjoIw8mwKglsHOHCMANcDX/xKr7TTzUw5sJXvgNe0yglFlAhxgSxDkMMjTl3
Z6a7eaiDRRwxcz5jovjtGYxPFXzERN1XQohHTXUtmfIhAmRX5+BXLiB8Vqmhh8Zz
GoW4sBBy8LMRIqC97yashjEAX7bcVdU2Q/smIWDimYsOHrg3mb9jvyimZ7PruL8R
EEnR6uXcy2ckAdMAyOnJ1VU3UMXr2UN3LZ8NKmUQiLevNJeUyDqWzIragkKqCd0o
1GAYQjATERhjwzQMmVA4AOP2T9BMmwVX/T5ZA67acmDxiA==
-----END CERTIFICATE-----
'';
  csKey = ''
-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAoCzoSkYev+OHX5VqKK+W1Z6h4HG9kiDAT2R1+Ro18u1dQhvL
A8tdv3JxxrLlUIvT7TPzxXtcysTT5BYj6fNUTcMxkgZHz9z9GObeIy6SRK+1fo2z
9GoOn0B26q5QdtBNFMk+XkwvvM85JpZsGY0rmYpCdihB3JMjRG9W/417Bd6YRjpM
t6gmcDGmOZ40CBclgv/TEWv4vw38fqmy9NpqmlylCYQeDhMWn+MVj/FOk25bjrYc
7meAXzHaz9sGJlIz/MQE9FRjasYuFA7CTfMXuPrA8wSGKF1qjtBucD8plKtE0ZUd
dwtXb12x9n7MmtIxeGiW0EUjCkPpofxALNmHSQIDAQABAoIBABuGXAyXbCVRdivo
wytmsSbYcbzeDtOTqTh7bQJ3jJnITGRV3lcylVOW2RJqH5ntzWdPrC5dep6loDvr
yhQj6nLKfjQ3vBNuSFgFJFsrX5tKDohG1YvExep763N8rPsd5IET7BHMSc/KVGnb
I4xog/uIlM81L8w1xLO35l1X9LIXPrzKr1XTAKPjSwe7hfdV8vDxI7l7xC8jkSNZ
vcx7x4kXPMgtgaaVADErzY7wS+J/H3GLCM0SZ6fqht0zFeZf+NyUauzak9cx6cIP
h/UDV7gCYV3VMtQfD30FtttfKWvVlNA5fxqBbD2Dxk1grKOpfYLMMDN1jw2SLlu2
ekGnr9kCgYEAx0TpOAjDb0/TYS54nTDm8/N/QJbEj/JNrw7v4MVwVQtC2tlAxbtO
MRtk2zkrUtCFCH/esy8yKQDScaJUFAW8QgBC0e7a+oikp7jrdvIx1gUTSE51TslZ
aOpAqsFpaMqFnW1OaeAtu3S2+CMDwnAed8irrpp/IMNgJDfx1w/f/KcCgYEAzcbG
JObZqEkPDY5LPTbX8cGZpwOZUtIfFfOAgxM8tkuGlbFHiA6bGUuqOBgstdJkKnHZ
IgElr4NrqwOQ99KHgWiKTZaOiLZbJlbtDpulU3Kpv0lwg5T7d+Oi6CostH35cxGS
mAn3DSji7Cwn7wSZ94JT3aASDmPGoZeTQPd4Ko8CgYBzAyEgyF4Upxw34RyYjZsf
fpEZ9GsrMg0IVzS4pPx6+W7y5aXu+nbc/RSvO0X4HIZMK5GcFkd7RxAvqiOhEtZf
ucrXZGdbZvayH5c4Jf4BqxhACZjHiotidKIybEOsygdon6g8j7mVkn3wpjULSq8r
L9V3h5CMlnetL+UT3gPHzQKBgQCAzyy5bMhSz2jc03XFm88RRl8obNhNP7q1wvdv
FVurwRs+GPrt8DamXvbupjNWnZyV9S42WwF8HIgJRPI6L08jco0ghF40tfHYzhEW
U9fppJ0dYJtNwrSnF5eiPMQ/N5wuq5FYGuTLGAz0Sa+1ruuyQ6K72Ld0yoBMJtXG
lSJjgQKBgQC27DY/hQmC6psbCyS7tYDZfasPoigeQJ3wM5wtoOVsjy/qc2sBeJp4
hnjIiGz7Iq3wUpJXCnh5Xo7aOq4/fii7pBv2mCe22QGuppqZoZOtpEElWSQrqDll
42cJ3aFElm7NhOigb7bXzuszLDrcPNPLnKcDwH6eEeegx4Tb09CKaA==
-----END RSA PRIVATE KEY-----
'';
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
        node.wait_until_succeeds(f"ping -c 1 {addrs[i]}")
    '';
  });
}
