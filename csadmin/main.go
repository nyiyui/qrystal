package csadmin

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/nyiyui/qrystal/util"
)

type command struct {
	FS      *flag.FlagSet
	Handler func()
}

var commands = map[string]command{}

var serverAddr *string
var ct *util.Token
var certPath *string

func Main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "qrystal-cs-admin: remotely control Qrystal CS instances\n")
		flag.PrintDefaults()
	}

	serverAddr = flag.String("server", "", "server address")
	ctRaw := flag.String("token", "", "central token")
	certPath = flag.String("cert", "", "path to server cert (leave blank to use system certs)")

	flag.Parse()
	if len(flag.Args()) == 0 {
		log.Print("expected a subcommand")
		flag.Usage()
		os.Exit(1)
	}
	var err error
	ct, err = util.ParseToken(*ctRaw)
	if err != nil {
		log.Fatalf("parse token: %s", err)
	}

	cmd, ok := commands[flag.Arg(0)]
	if !ok {
		log.Printf("unknown command %s", flag.Arg(0))
		flag.Usage()
		os.Exit(1)
	}
	err = cmd.FS.Parse(flag.Args()[1:])
	if err != nil {
		log.Fatal(err)
	}
	cmd.Handler()
}
