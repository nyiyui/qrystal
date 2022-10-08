package node

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

var outboundRe = regexp.MustCompile(`(dev )(?P<dev>\S+)`)

func getOutbound() (string, error) {
	errBuf := new(bytes.Buffer)
	outBuf := new(bytes.Buffer)
	// NOTE: workaround for $PATH being weird
	cmd := exec.Command("/usr/bin/ip", "route", "show", "default")
	cmd.Stderr = errBuf
	cmd.Stdout = outBuf
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("exec: %w\n\n%s", err, errBuf)
	}
	out := outBuf.String()
	// NOTE: check if locale affects this
	if !strings.HasPrefix(out, "default via") {
		return "", errors.New("unexpected output not starting with \"default via\"")
	}
	return outboundRe.FindStringSubmatch(out)[2], nil
}

func makePost(flag, rule, cnn, outbound string) string {
	return fmt.Sprintf(
		`iptables %s FORWARD %s -i %s -j ACCEPT && iptables -t nat %s POSTROUTING %s 0 -o %s -j MASQUERADE`,
		flag,
		rule,
		cnn,
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
