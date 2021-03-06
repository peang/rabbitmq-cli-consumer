// +build integration

package main_test

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"testing"
)

// TestHelperProcess is used as executable during integration tests.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	defer os.Exit(0)
	log.SetFlags(0)

	flags, args := flagsAndArgs()
	outputFile := flags.String("output", "./command.log", "the output file")
	isCompressed := flags.Bool("comp", false, "whether the argument is compressed or not")
	isPipe := flags.Bool("pipe", false, "whether the argument passed via stdin (TRUE) or argument (FALSE)")
	flags.Parse(args)

	f, err := outputWriter(*outputFile)
	if err != nil {
		log.Fatalf("failed to open output file: %v\n", err)
	}
	defer f.Close()

	writeLine(f, []byte("Got executed"))

	first, second, err := payload(*isPipe, *isCompressed, args)
	if err != nil {
		log.Fatalln(err)
	}

	writeLine(f, first)
	writeLine(f, second)
}

func flagsAndArgs() (*flag.FlagSet, []string) {
	args := os.Args
	for len(args) > 0 {
		if args[0] == "--" {
			args = args[1:]
			break
		}
		args = args[1:]
	}

	return flag.NewFlagSet(os.Args[0], flag.ExitOnError), args
}

func outputWriter(file string) (*os.File, error) {
	if file == "-" {
		return os.Stdout, nil
	}

	return os.Create(file)
}

func payload(isPipe, comp bool, args []string) ([]byte, []byte, error) {
	if !isPipe {
		return payloadArgument(comp, args)
	}

	return payloadPipe()
}

func payloadArgument(comp bool, args []string) ([]byte, []byte, error) {
	first := []byte(args[len(args)-1])
	second, err := decode(first, comp)
	if err != nil {
		return nil, nil, err
	}

	return first, second, nil
}

func payloadPipe() ([]byte, []byte, error) {
	pipe := os.NewFile(3, "/proc/self/fd/3")
	defer pipe.Close()
	first, err := ioutil.ReadAll(pipe)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read metadata from pipe: %v", err)
	}

	second, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read body from pipe: %v", err)
	}

	return first, second, nil
}

func writeLine(w io.Writer, p []byte) (int, error) {
	n, err := w.Write(p)
	if err != nil {
		return n, err
	}

	m, err := w.Write([]byte("\n"))

	return n + m, err
}

func decode(body []byte, comp bool) ([]byte, error) {
	var r io.Reader
	r = base64.NewDecoder(base64.StdEncoding, bytes.NewBuffer(body))
	if comp {
		zr, err := zlib.NewReader(r)
		if err != nil {
			return nil, fmt.Errorf("failed to create zlib reader: %v", err)
		}
		defer zr.Close()
		r = zr
	}

	return ioutil.ReadAll(r)
}
