package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "toxipacket",
		Usage: "Simulate network inconsistency",
		Commands: []*cli.Command{
			{
				Name:  "add",
				Usage: "Add a rule to an interface",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "ip",
						Value: "127.0.0.1",
						Usage: "Target ip",
					},
					&cli.StringFlag{
						Name:    "port",
						Aliases: []string{"p"},
						Usage:   "Target port",
					},
					&cli.StringFlag{
						Name:    "loss",
						Aliases: []string{"l"},
						Usage:   "Packet loss to be applied",
					},
				},
				Action: func(context *cli.Context) error {
					iface, err := getInterfaceForIP(context.String("ip"))
					if err != nil {
						return cli.Exit(err, 1)
					}

					err = applyTCRules(iface, context.String("ip"), context.Int("port"), context.Int("loss"))
					if err != nil {
						return cli.Exit(err, 1)
					}
					return nil
				},
			},
			{
				Name:    "remove",
				Aliases: []string{"rm"},
				Usage:   "Remove a rule ",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "ip",
						Value: "127.0.0.1",
						Usage: "Target ip",
					},
				},
				Action: func(context *cli.Context) error {
					iface, err := getInterfaceForIP(context.String("ip"))
					if err != nil {
						return cli.Exit(err, 1)
					}

					err = removeTCRulesFromInterface(iface)
					if err != nil {
						return cli.Exit(err, 1)
					}
					return nil
				},
			},
			{
				Name:  "show",
				Usage: "Show the currently applied rules",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "ip",
						Value: "127.0.0.1",
						Usage: "Target ip",
					},
				},
				Action: func(context *cli.Context) error {
					iface, err := getInterfaceForIP(context.String("ip"))
					if err != nil {
						return cli.Exit(err, 1)
					}

					output, err := getActiveRules(iface)
					if err != nil {
						return cli.Exit(err, 1)
					}
					fmt.Println(output)
					return nil
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func getActiveRules(iface string) (string, error) {
	output, err := exec.Command("sudo", "tc", "qdisc", "show", "dev", iface).CombinedOutput()
	return string(output), err
}

func getInterfaceForIP(ip string) (string, error) {
	// Check if the IP is localhost
	if ip == "127.0.0.1" {
		return "lo", nil
	}

	// Check if it's a valid IP address
	if net.ParseIP(ip) == nil {
		return "", fmt.Errorf("invalid IP address: %s", ip)
	}

	cmd := exec.Command("ip", "route", "get", ip)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// Parse the output to extract the interface name
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		for i, field := range fields {
			if field == "dev" && i+1 < len(fields) {
				return fields[i+1], nil
			}
		}
	}

	return "", fmt.Errorf("could not determine interface for IP %s", ip)
}

func applyTCRules(iface, ip string, port int, loss int) error {
	if loss <= 0 || loss > 100 {
		return fmt.Errorf("Loss needs to be in range 0-100")
	}

	output, err := exec.Command("sudo", "tc", "qdisc", "add", "dev", iface, "root", "handle", "1:", "prio").CombinedOutput()
	if err != nil {
		if strings.HasPrefix(string(output), "Error: Exclusivity flag on, cannot modify") {
			return fmt.Errorf("A rule for this ip already exists")
		}
		return fmt.Errorf(string(output))
	}

	// Add filter
	var filterCmd *exec.Cmd
	if port > 0 {
		filterCmd = exec.Command("sudo", "tc", "filter", "add", "dev", iface, "protocol", "ip", "parent", "1:", "prio", "1", "u32", "match", "ip", "dst", ip, "match", "ip", "dport", strconv.Itoa(port), "0xffff", "flowid", "2:1")
	} else {
		filterCmd = exec.Command("sudo", "tc", "filter", "add", "dev", iface, "protocol", "ip", "parent", "1:", "prio", "1", "u32", "match", "ip", "dst", ip, "flowid", "2:1")
	}
	output, err = filterCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf(string(output))
	}

	// Add netem qdisc
	output, err = exec.Command("sudo", "tc", "qdisc", "add", "dev", iface, "parent", "1:1", "handle", "2:", "netem", "loss", fmt.Sprintf("%d%%", loss)).CombinedOutput()
	if err != nil {
		return fmt.Errorf(string(output))
	}

	// Print success message
	if port > 0 {
		fmt.Printf("Applied %d%% packet loss to %s:%d on interface %s\n", loss, ip, port, iface)
	} else {
		fmt.Printf("Applied %d%% packet loss to %s on interface %s\n", loss, ip, iface)
	}

	return nil
}

func removeTCRulesFromInterface(iface string) error {
	cmd := exec.Command("sudo", "tc", "qdisc", "del", "dev", iface, "root")

	output, err := cmd.CombinedOutput()
	if err != nil {
		if strings.HasPrefix(string(output), "Error: Cannot delete qdisc with handle of zero") {
			return fmt.Errorf("No rules to remove")
		}
		return fmt.Errorf("%s", string(output))
	}
	return nil
}
