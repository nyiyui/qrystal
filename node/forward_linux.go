package node

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

var CommandIp string
var CommandIptables string

var outboundRe = regexp.MustCompile(`(dev )(?P<dev>\S+)`)

func getOutbound() (string, error) {
	errBuf := new(bytes.Buffer)
	outBuf := new(bytes.Buffer)
	// NOTE: workaround for $PATH being weird
	cmd := exec.Command(CommandIp, "route", "show", "default")
	cmd.Stderr = errBuf
	cmd.Stdout = outBuf
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("exec: %w\n\n%s", err, errBuf)
	}
	out := outBuf.String()
	// NOTE: check if locale affects this
	if !strings.HasPrefix(out, "default via") {
		return "", fmt.Errorf("unexpected output (%s) not starting with \"default via\"", out)
	}
	return outboundRe.FindStringSubmatch(out)[2], nil
}

func makePost(flag, rule, cnn, outbound string) string {
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

func makePostUp(cnn, outbound string) string {
	return makePost("-A", "", cnn, outbound)
}

func makePostDown(cnn, outbound string) string {
	return makePost("-I", "0", cnn, outbound)
}
