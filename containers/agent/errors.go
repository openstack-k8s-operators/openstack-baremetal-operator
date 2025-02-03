package main

import (
	"errors"
)

var (
	ErrOSImageNotFound = errors.New("could not find any OSImage checksum file")
)
