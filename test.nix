{ self, system, nixpkgsFor, nixosLibFor, ... }:
let
  node1Token = "3ztQRDsLo+iOEeU8BJp7GTiAhrpMr8rLt5HrAlDUwNEItGhcjW98lsCyKIpmCT+AtkC1vLDkfRSvWk1JQlMVlw==";
  node1Hash = "72d487c5632716c8cdf3cf440ed29e14171e27c245c715f92e2517aee605fc71";
  node2Token = "oiiOXY3dRRgmm/DjPQFJja5OftVBrZffRpxyUWk2CWabdAn9jUfeIh4vlSE35eJK0qjfb7w/2/XJya/xoduNew==";
  node2Hash = "b7ba9038ed61f27a1e4142007eaaed7a0309fed9180bace0474d2891bada9395";
  rootCert = ''
-----BEGIN CERTIFICATE-----
MIIDSzCCAjOgAwIBAgIIIvlJ6cCSMx0wDQYJKoZIhvcNAQELBQAwIDEeMBwGA1UE
AxMVbWluaWNhIHJvb3QgY2EgMjJmOTQ5MCAXDTIyMTIxODE5MjYxN1oYDzIxMjIx
MjE4MTkyNjE3WjAgMR4wHAYDVQQDExVtaW5pY2Egcm9vdCBjYSAyMmY5NDkwggEi
MA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQC4FZle4vjdnJX0SUYmDj7OSxh0
tOH8fyFe/WhETN+0EKfFeRxckExBDLIzkcjpdNGWMJfZE5ODvFxYSUZrorhQNpBU
9+Ldwo6vm381/5lV67BlVop5545rXUdZmEYdzrMkvIK2knxx2DJEiLKmO8grkrFu
rDxq0N/7YfsqtAr+qqa6RihkJnFAunnXMNwelCK22tjUf0ybIMjrx9HfvgcVhKjg
V3uJrgP+IH0L2kXTAZk6aUS8BY4MpfiTGlgNjZepok46fKf55OJKZIR6YNARtfy6
u+OX7ngls/VK6pzFPdcfVf+mmpOKf3TKyQjQA2T0QjYyFwrEmmYGrMRQMIJrAgMB
AAGjgYYwgYMwDgYDVR0PAQH/BAQDAgKEMB0GA1UdJQQWMBQGCCsGAQUFBwMBBggr
BgEFBQcDAjASBgNVHRMBAf8ECDAGAQH/AgEAMB0GA1UdDgQWBBSjCcxWVdVtcWck
j19JrTJ+6aArLDAfBgNVHSMEGDAWgBSjCcxWVdVtcWckj19JrTJ+6aArLDANBgkq
hkiG9w0BAQsFAAOCAQEAkdVKhwkpeRh5771K5rZ/Nnc8MBkWAFtfnNso4iu1R1tm
eBS1+5+MtTr09x9j6/MHBpXhQ9RiXlmPeZ8RxrMV41tpOjSWa1K8rbeQMg4/P7RW
mpvrFfbs3yvoAL9Ge3A0De7HK4mnkgPsStGv3lkLbc7daQj7KZvSlVKe96CDpFVs
Sy0ShTI9qTQ5ef1yBdgXduxcEyqUkKz0nIxLrxKgn6SR7gHYuEhHPFTIV/0nuTeK
e10PhLbzCiijUbXIjXDItDrt94mBssxgOMTwJDi54ovaEkGtc1MfM1tyyqmWqz5o
BQ1yF+Ass/zzbui911XnI+NRM+93xAg5bRUgJwnKwA==
-----END CERTIFICATE-----
'';
  rootKey = ''
-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAuBWZXuL43ZyV9ElGJg4+zksYdLTh/H8hXv1oREzftBCnxXkc
XJBMQQyyM5HI6XTRljCX2ROTg7xcWElGa6K4UDaQVPfi3cKOr5t/Nf+ZVeuwZVaK
eeeOa11HWZhGHc6zJLyCtpJ8cdgyRIiypjvIK5Kxbqw8atDf+2H7KrQK/qqmukYo
ZCZxQLp51zDcHpQittrY1H9MmyDI68fR374HFYSo4Fd7ia4D/iB9C9pF0wGZOmlE
vAWODKX4kxpYDY2XqaJOOnyn+eTiSmSEemDQEbX8urvjl+54JbP1SuqcxT3XH1X/
ppqTin90yskI0ANk9EI2MhcKxJpmBqzEUDCCawIDAQABAoIBADUbbhrUyk1M7moC
da1m8LGdMpoA0S2CE8OOwfTqZKNTJsOutAL0Ujt2CTcdeOP5Irn8nOIwZp9byRxj
T2CgGiJyC2On/BhUF8wLxUBz0+3YyBQESoDuz8SjrYDokFnrFv2jMOaxDhvd7mqd
MUUJ/C6t7GhsYiXCysuAMfDY7k8XuJ2o6L06yv4AuIzamyE+BmTSdhx/rMw1Yavs
6QK9xpkbsQx7XocbmC3bPaz1RJoGazpaUO8BbsBho9MTzzWvh87dHxUTte15zB7Y
tgpjd1tegTSLRrx7tKxo92KuXTWxoLCU1H1ooq+sbnLstCT3ofbSYS1QdcRIuyau
6ZQNUfECgYEAxWvet+iMAhO8CfMoIRoyMspJJmY0jdi4FUBUyEdWpzbVnbafTFYt
Ig0OWpC6PvEg8DFt9hOu1Un8oH7oWrtHMhw0ZLA5xpTJ1QPJObhLeM7LxZV5puvU
VIIDHrwadexQH0xlrEiITPLOQkrMHsTTS4Gd88NAAdi0i9IyimL8G/MCgYEA7rSm
lZgOmqfDu96FbwvlKGo1V4KzF/ZVEg14haO9WUAZj3It/tR6dfRUCj3id3aPQS/L
GeNT0Fzxq/Z3ayactQ3DFE9kbrIHp5GGhfQX1e4P1c5dxgLmmqCDhcv5+miQCx51
JtTQrGG73XbVNtHZY/O6GXcXOgjMhtju1DAUdakCgYEAv3T2YDp5FUaYNLoIr9mc
1x7QRBoYW3vSQmHKFxUAF1gZYEMMR9bHHF+3DOOQi5wDSo1VS7EY+6YuBmQs6Fj5
GcK6mO9CiLAg8KEkVALDxpweiDaG7PeGSpJvfi4EJ1qO9Vt8utD4xk8u8qFhRXGy
TGaejRlMiL3lkje+ZfDK+DsCgYBrNhIX4Eq25aDA8YmmvYX4J/O7UUWU/pto10oJ
Y+h4fJS+W78S1GYIMmvIidD8bPCci5XCE9siG4yj+rfaFWaO3xZ+OcZW/Xj4pyDv
axmFiT3tfpmZhNYEHxHTdzDYajw/8jcV8MGkmuTg7C2JSKlF/kLYiyeQdkE+U5K5
FLsruQKBgQCVCdZ+GrtNzL7Sxw4dcHBb79TxtnVVFyYNBAVV2iDvpgFu58ie4TND
w+7VcYbhQP6ala5OaEXO6C0xHpETHFVTeM/pARx78lc1lMm5NGJ3Gd+cwXxQ8R33
iT1ipC9WqJpcaEp9a6iYyKpOiMEURW9sILpOvAbGisGSDb4Z+c7DOQ==
-----END RSA PRIVATE KEY-----
'';
  csCert = ''
-----BEGIN CERTIFICATE-----
MIIDSDCCAjCgAwIBAgIIQRvAzz9DI9MwDQYJKoZIhvcNAQELBQAwIDEeMBwGA1UE
AxMVbWluaWNhIHJvb3QgY2EgMjJmOTQ5MB4XDTIyMTIxODE5MjYxN1oXDTI1MDEx
NzE5MjYxN1owITEfMB0GA1UEAxMWY3MudGVzdG5ldC5leGFtcGxlLm5ldDCCASIw
DQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBALUnOtOm1Kje6QGFmbDd1B2d2sBN
Hh3ZJDGBNLvFNp/FkDccTHi8qSXlg49BM+qq216UWA629ow652K/8v17UODyJdQV
QJU7DgRxJrA5A5BYPA53j22Yz4Q417p6mwgVCsPDqgG/xDP4nnlwryTPz/hTRIRt
jvldkfy92j504sEw9nVF7YYyfQfvdnMW5PRi+f4mwIobZwsjFT0NzjS5wj5frERs
Wdv8uyQ5eJGZmu6kd9Uqg1AIZPZ8ZUwV/mG1LUNrylZ+XjOn4ImmHahFQfJvWqXZ
bo4YIBJSE0wltS+H0b6bJBYq4pHkuM5tKuaCmZdb2NCWqzYXZTV0tks6o1cCAwEA
AaOBhDCBgTAOBgNVHQ8BAf8EBAMCBaAwHQYDVR0lBBYwFAYIKwYBBQUHAwEGCCsG
AQUFBwMCMAwGA1UdEwEB/wQCMAAwHwYDVR0jBBgwFoAUownMVlXVbXFnJI9fSa0y
fumgKywwIQYDVR0RBBowGIIWY3MudGVzdG5ldC5leGFtcGxlLm5ldDANBgkqhkiG
9w0BAQsFAAOCAQEAbp7ugJuzSpUToKm0rSZZ8XkJ1MahQ4OsM+vGcN6x2TZZrWUs
d4jHOlbGaZnu9wQUzAjGA/CkCIwB2qdcTZhae95tg/H1UUlJIcTRLhp/Y6IwKcul
T4SsuF5qyO3q3boaI+1bsDnjhXaclsaHJpo6uaU9257LEa3xmPt2r1/YSg/oa5fV
QMDoUVvqtuHqIIFscObp3bly3ZIwC72Chj8h8Ys09i4QiAyr81eAP+4IruVxJjOC
SqE65bF2dHmuG5qOlDcCl5aeXGzt4qY8emJo4d65rSxu8OLwVIG0t1KkXtAdUZ/O
1m/aCLEDg8M1aDkVWwezne1NRnnFLadW0cY/og==
-----END CERTIFICATE-----
'';
  csKey = ''
-----BEGIN RSA PRIVATE KEY-----
MIIEogIBAAKCAQEAtSc606bUqN7pAYWZsN3UHZ3awE0eHdkkMYE0u8U2n8WQNxxM
eLypJeWDj0Ez6qrbXpRYDrb2jDrnYr/y/XtQ4PIl1BVAlTsOBHEmsDkDkFg8DneP
bZjPhDjXunqbCBUKw8OqAb/EM/ieeXCvJM/P+FNEhG2O+V2R/L3aPnTiwTD2dUXt
hjJ9B+92cxbk9GL5/ibAihtnCyMVPQ3ONLnCPl+sRGxZ2/y7JDl4kZma7qR31SqD
UAhk9nxlTBX+YbUtQ2vKVn5eM6fgiaYdqEVB8m9apdlujhggElITTCW1L4fRvpsk
FirikeS4zm0q5oKZl1vY0JarNhdlNXS2SzqjVwIDAQABAoIBAA/V8xWHcvWkLtg8
Npg4fA9uui2vUB+p2LkfI136unCzE41Nwv2W+G5gpuSB/ajY8L5O13fJ1LmjeJCw
WOyBuCtB376vcOras7n9rjUfdslKfU2CdB5Pimxzj6A0kZLeTAea9iSa/+rPJANX
r2fXZsW9ebLd5O61mEpwykBFdYEPwgFdE7oSTo600ahI0dJZ2JObnn66tMUDT1lr
A8Wzo1SehRWWQA6XdHFhvOduA5b5djYW79fC09Yh3U7dnLpXu5CTMSk5LEL3iS2i
IGncxu5bWGY0dXmZgLetsfrC1W2v/8pA2kkzWnx14aQ+uXQ8T44bUeDrctjcmxqN
7Pnf48kCgYEA0WVzOfqYrwgvvcU06Wb/1EeYlThN9SP/PZylPyJJp3+fz2Uecjuv
pQvPOSdtlweyF0GnKh6xtgF2s4aHnIeXyVYEIkdIP3Jzvfak6MjBYfBgn4jQG13h
kdCd4kHbL5TMsjj6Amuh0Ih0XULXyu2VDVnmy7xRf3zSQnjpl5/8npsCgYEA3Xib
BRa0GJPc1viTG+WFVaPHZ/EBhnhzrZcZAt0FZTkxYDJQYN0MvQur92t4sZfOtdrp
uYCUaUuLr1N6pk+0eWg1wVUg34Tuxw1LM+P5oQp9mupqK+hcWFBXkzFPNJE9Fxwk
7Y+mteekAkJUGoilKuXKfUW4x/kaxqI5Aoa9m/UCgYBSYCzCZFlokjnl2A0GvSRr
qHbYTTwt8ilZXaSMf7qmEEkYV9lwaxagQVMWUvKD9d0T1RokMcsLpOvDmGsFIzqN
VC9wJMbBXw81bjBV+5RIKT55xGLKQVaZ/I4AEpRd1ZXpjwyboygXV3cfsUofZPO8
Ot/WypDtLHey+so6gg/pfQKBgCjhNzQUQb/7oxrnHThcAGWTap5MBS0OFMQpDMvT
gkhx6yRHhUCr7MsEWYS9CLU3QUeeFeBQ1JQvBqShMxV5xuVWD/4UuZGolu6VDJmS
biSErDSpKlnadRk0E0YvJuCcInuejU5wYqRXEpX8KkwPhvVJHzxKX1ZCK+gYT4+g
0WT1AoGAWANSu0Z+JlzXJ4jsyPSM/wT+yJAx4Vx6hT+K/M6LeErd/GVITrprbRdi
FrPsI0Vtbe9DTzjmBF2RrRzZ8ePirbMlETPXkS4v4sISIhmWnPi3cy4GjuxQ8tfq
V3q3hlKpuGWhFQxyV4E3CqkHfP1jhp7xdZgrpi5uiznUGxrU9j0=
-----END RSA PRIVATE KEY-----
'';
in
{
  cs = let
    pkgs = nixpkgsFor.${system};
    lib = nixosLibFor.${system} { inherit system; };
  in
    lib.runTest {
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
                { name = "node1"; hash = node1Hash; can.pull = true; }
                { name = "node2"; hash = node2Hash; can.pull = true; }
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
  one = let
    pkgs = nixpkgsFor.${system};
    lib = nixosLibFor.${system} { inherit system; };
    networkName = "testnet";
  in let
    node = (token: { pkgs, ... }: {
      imports = [ self.outputs.nixosModules.${system}.node ];

      networking.firewall.allowedTCPPorts = [ 39251 ];
      qrystal.services.node = {
        enable = true;
        #config.css = [
        #  {
        #    comment = "cs";
        #    endpoint = "cs.testnet.example.net";
        #    tls.certPath = rootCert + "\n" + csCert;
        #    networks = [ networkName ];
        #    inherit token;
        #  }
        #];
      };
    });
    base = { # TODO
      virtualisation.vlans = [ 1 ];
    };
  in
  lib.runTest ({
    name = "integration";
    hostPkgs = pkgs;
    nodes = {
      node1 = node node1Token;
      node2 = node node2Token;
      cs = { pkgs, ... }: {
        imports = [ self.outputs.nixosModules.${system}.cs ];

        networking.firewall.allowedTCPPorts = [ 39252 ];
        qrystal.services.cs = {
          enable = true;
          config = {
            tls.certPath = builtins.toFile "testing-insecure-cert.pem" csCert;
            tls.keyPath = builtins.toFile "testing-insecure-key.pem" csKey;
            tokens = [
              { name = "node1"; hash = node1Hash; can.pull = true; }
              { name = "node2"; hash = node2Hash; can.pull = true; }
            ];
            central.networks.${networkName} = {
              keepalive = "10s";
              listenPort = 58120;
              ips = [ "10.123.0.1/16" ];
              peers.node1 = { allowedIPs = [ "10.123.0.1/16" ]; };
              peers.node2 = { allowedIPs = [ "10.123.0.2/16" ]; };
            };
          };
        };
      };
    };
    testScript = { nodes, ... }: ''
      nodes = [node1, node2]
      start_all()
      cs.wait_for_unit("qrystal-cs.service")
      list(map(lambda node: node.wait_for_unit("qrystal-node.service"), nodes))
      addrs = ["10.123.10.1", "10.123.10.2"]
      for node in nodes:
        for addr in addrs:
          node.succeed(f'ping {addr}')
    '';
  });
}
