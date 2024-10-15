package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
)

func main() {
	// Define flags
	ip := flag.String("ip", "127.0.0.1", "Target ip")
	remove := flag.Bool("remove", false, "Remove rules for ip")
	port := flag.Int("p", 0, "Port to apply rules to")
	loss := flag.Int("loss", 0, "Packet loss percentage")

	// Parse flags and remaining arguments
	flag.Parse()
	args := flag.Args()

	if len(args) < 1 {
		fmt.Println("No options specified")
		fmt.Println("Usage: ./toxipacket <ip_address> [flags]")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// I think it's a nice to have? Might change later
	if *ip == "" || *ip == "localhost" {
		*ip = "127.0.0.1"
	}

	iface, err := getInterfaceForIP(*ip)
	if err != nil {
		fmt.Printf("Error determining network interface: %s\n", err)
		os.Exit(1)
	}

	if *remove {
		err := removeTCRulesFromInterface(iface)
		if err != nil {
			fmt.Printf("Error while removing tc rules from interface: %s\n", err)
		}
	} else {
		applyTCRules(iface, *ip, *port, *loss)

	}
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
	commands := []string{
		fmt.Sprintf("tc qdisc add dev %s root handle 1: prio", iface),
	}

	// Add port-specific filter if a port is specified
	if port > 0 {
		commands = append(commands, fmt.Sprintf("tc filter add dev %s protocol ip parent 1: prio 1 u32 match ip dst %s match ip dport %d 0xffff flowid 2:1", iface, ip, port))
	} else {
		commands = append(commands, fmt.Sprintf("tc filter add dev %s protocol ip parent 1: prio 1 u32 match ip dst %s flowid 2:1", iface, ip))
	}

	commands = append(commands, fmt.Sprintf("tc qdisc add dev %s parent 1:1 handle 2: netem loss %d%%", iface, loss))

	for _, cmd := range commands {
		err := executeCommand("sudo", "sh", "-c", cmd)
		if err != nil {
			fmt.Printf("Error executing command: %s\n", err)
			os.Exit(1)
		}
	}

	if port > 0 {
		fmt.Printf("Applied %df%% packet loss to %s:%d on interface %s\n", loss, ip, port, iface)
	} else {
		fmt.Printf("Applied %d%% packet loss to %s on interface %s\n", loss, ip, iface)
	}

	return nil
}

func executeCommand(name string, arg ...string) error {
	cmd := exec.Command(name, arg...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
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
