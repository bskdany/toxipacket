package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func main() {
	// Define flags
	ip := flag.String("ip", "127.0.0.1", "Target ip")
	remove := flag.Bool("remove", false, "Remove rules for ip")
	port := flag.Int("p", 0, "Port to apply rules to")
	loss := flag.Int("loss", 0, "Packet loss percentage")
	show := flag.Bool("show", false, "Show active rules")

	// Parse flags and remaining arguments
	flag.Parse()
	// args := flag.Args()

	if *remove == false && *loss == 0 && *show == false {
		fmt.Println("Usage: ./toxipacket <ip_address> [flags]")
		flag.PrintDefaults()
		os.Exit(1)
	}

	if *loss < 0 || *loss > 100 {
		fmt.Println("Loss needs to be in range 0-100")
		os.Exit(1)
	}

	// I think it's a nice to have? Might change later
	if *ip == "localhost" {
		*ip = "127.0.0.1"
	}

	iface, err := getInterfaceForIP(*ip)
	if err != nil {
		fmt.Printf("Error determining network interface: %s\n", err)
		os.Exit(1)
	}

	if *show {
		output, err := getActiveRules(iface)
		if err != nil {
			fmt.Printf("Error getting active rules. %s\n", err)
			os.Exit(1)
		}
		fmt.Println(output)
	} else {
		if *remove {
			err := removeTCRulesFromInterface(iface)
			if err != nil {
				fmt.Printf("Error while removing rules: %s\n", err)
				os.Exit(1)
			}
		} else {
			err := applyTCRules(iface, *ip, *port, *loss)
			if err != nil {
				fmt.Printf("Error while applying rules. %s\n", err)
				os.Exit(1)
			}
		}

	}
}

func getActiveRules(iface string) (string, error) {
	output, err := exec.Command("sudo", "tc", "qdisc", "show", "dev", iface).CombinedOutput()
	return string(output), err
}

func getInterfaceForIP(ip string) (string, error) {
	// Check if the IP is localhost
	if ip == "127.0.0.1" || ip == "::1" {
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
