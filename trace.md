# Execution Trace (Node)

Node can provide an execution trace (using [Go's `runtime/trace`](https://pkg.go.dev/runtime/trace)) from startup until a set of CNs are initially reified (i.e. the WireGuard networks for the CNs are configured and Hokuto DNS is configured for them, if applicable).

## Configuration

Set the `QRYSTAL_TRACE_OUTPUT_PATH` envvar for the output path of the trace. Note that the NixOS modules set `PrivateTmp=yes` on the systemd unit (e.g. if you set `QRYSTAL_TRACE_OUTPUT_PATH=/tmp/qrystal-trace`, then the trace would be in something like `/etc/systemd-private-abc-qrystal-node.service-def/tmp/qrystal-trace`).

Set the `QRYSTAL_TRACE_UNTIL_CNS` envvar to a JSON list of the CNs. Once all of the CNs specified in here are reified, the trace will be stopped (and saved).
Example: `["examplenet","othernet"]`
