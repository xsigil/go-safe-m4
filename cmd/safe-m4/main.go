package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/xsigil/go-safe-m4/pkg/engine"
)

func main() {
	sandboxRoot := flag.String("sandbox", ".", "Sandbox root directory for include directives")
	flag.Parse()

	var input []byte
	var err error

	args := flag.Args()
	if len(args) > 0 {
		input, err = os.ReadFile(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "safe-m4: error reading file: %v\n", err)
			os.Exit(1)
		}
	} else {
		input, err = io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "safe-m4: error reading stdin: %v\n", err)
			os.Exit(1)
		}
	}

	eng := engine.NewEngine(engine.WithSandboxRoot(*sandboxRoot))
	output, err := eng.Expand(string(input))
	if err != nil {
		fmt.Fprintf(os.Stderr, "safe-m4: expansion error: %v\n", err)
		os.Exit(1)
	}

	fmt.Print(output)
}
