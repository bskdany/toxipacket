## Toxipacket
Simulate network instability the easy way. It is a wrapper around the tc Linux utility, therefore the latter is the only supported platform

## How to use
```
NAME:
  toxipacket - Simulate network instability

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

## About
This cli tool was built in frustration of how complex it is to simulate packet loss. The only existing similar thing I found is [toxiproxy](https://github.com/Shopify/toxiproxy) but it's not exactly the quick and easy solution. 
