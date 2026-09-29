package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/mebyus/epox/internal/froll"
)

var lines = []string{
	"hello, world!",
	"quick brown fox jumps over a stick on a meadow",
	"very important line",
	"another test line",
	"long long long long line that just wont end right here",
	"test froll package with us much data as possible",
}

func main() {
	r, err := froll.NewRoller(&froll.Config{
		Path:    "data/test/froll.log",
		MaxSize: 1 << 24,
		MaxNum:  3,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "create roller: %v\n", err)
		os.Exit(1)
	}

	n := 1000000
	start := time.Now()
	for i := range n {
		k := i % len(lines)
		line := lines[k]
		nstr := strconv.FormatInt(int64(i), 10)

		_, err := io.WriteString(r, nstr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "write roller linenum: %v\n", err)
		}
		_, err = io.WriteString(r, " ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "write roller space: %v\n", err)
		}

		_, err = io.WriteString(r, line)
		if err != nil {
			fmt.Fprintf(os.Stderr, "write roller data: %v\n", err)
		}
		_, err = io.WriteString(r, "\n")
		if err != nil {
			fmt.Fprintf(os.Stderr, "write roller newline: %v\n", err)
		}
	}
	fmt.Printf("written %d lines over %v\n", n, time.Since(start))
}
