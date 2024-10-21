# toxipacket
Simulate network instability

```
NAME:
  toxipacket - Simulate network inconsistency

USAGE:
  toxipacket [global options] command [command options]

COMMANDS:
  add           Add a rule to an interface
    --ip:       Target ip (default 127.0.0.1)
    --port, -p: Target port
    --loss, -l: Packet loss to be applied
  remove, rm    Remove a rule
    --ip:       Target ip (default 127.0.0.1)
  show          Show the currently applied rules
    --ip:       Target ip (default 127.0.0.1)
  help, h       Shows a list of commands or help for one command

GLOBAL OPTIONS:
  --help, -h    Show help
```
