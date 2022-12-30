package node

import (
	"fmt"
	"os"
)

var CommandIp string
var CommandIptables string

func init() {
	if c := os.Getenv("NODE_COMMAND_IP"); c != "" {
		CommandIp = c
	}
	if c := os.Getenv("NODE_COMMAND_IPTABLES"); c != "" {
		CommandIptables = c
	}
}

var outbound = fmt.Sprintf(`%s route show default | grep -oP 'dev \K\W+'`, CommandIp)

func makePost(flag, rule, cnn string) string {
	return fmt.Sprintf(
		`%s %s FORWARD %s -i %s -j ACCEPT && %s -t nat %s POSTROUTING %s 0 -o %s -j MASQUERADE`,
		CommandIptables,
		flag,
		rule,
		cnn,
		CommandIptables,
		flag,
		rule,
		outbound,
	)
}

func makePostUp(cnn string) string {
	return makePost("-A", "", cnn)
}

func makePostDown(cnn string) string {
	return makePost("-I", "0", cnn)
}
