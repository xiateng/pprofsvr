package main

import (
	"flag"
	"strings"
)

// copy from https://github.com/YieldNull/webpprof

type GoFlags struct {
	*flag.FlagSet

	usageMsgs []string
	arguments []string
}

func NewGoFlags(args []string) *GoFlags {
	return &GoFlags{
		FlagSet:   flag.NewFlagSet("pprof", flag.ExitOnError),
		arguments: args,
	}
}

func (f *GoFlags) StringList(o, d, c string) *[]*string {
	return &[]*string{f.FlagSet.String(o, d, c)}
}

func (f *GoFlags) ExtraUsage() string {
	return strings.Join(f.usageMsgs, "\n")
}

func (f *GoFlags) AddExtraUsage(eu string) {
	f.usageMsgs = append(f.usageMsgs, eu)
}

func (f *GoFlags) Parse(usage func()) []string {
	f.FlagSet.Usage = usage
	_ = f.FlagSet.Parse(f.arguments)
	args := f.FlagSet.Args()
	if len(args) == 0 {
		usage()
	}
	return args
}
