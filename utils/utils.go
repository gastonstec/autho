// Copyright Kueski. All rights reserved.
// Use of this source code is not licensed

// Package utils provides some commonly used functions.
package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"
)

// Function IsJSON checks if the supplied string is in a
// valid JSON format. Returns true if the string is valid,
// false if not
func IsJSON(str string) bool {
	var js json.RawMessage

	err := json.Unmarshal([]byte(str), &js)
	if err == nil {
		return true
	}
	return false
}

// Function GetFunctionName returns the name of the current
// running function
func GetFunctionName() string {
	pc := make([]uintptr, 1)
	runtime.Callers(2, pc)
	f := runtime.FuncForPC(pc[0])
	return f.Name()
}

// Function GetOSEnv returns the operating system
// environment variables in a JSON format string.
func GetOsEnv() string {

	var env []string = os.Environ()

	envStr := "{\"OSEnvironment\": {"

	for index, value := range env {
		name := strings.Split(value, "=") // split by = sign
		name[1] = strings.ReplaceAll(name[1], "\\", "/")
		name[1] = strings.ReplaceAll(name[1], ":", ".")
		name[1] = strings.ReplaceAll(name[1], "\"", "")
		envStr = envStr + (fmt.Sprintf("\"%d\": \"%s=%s\",", index, name[0], name[1]))
	}

	envStr = strings.TrimRight(envStr, ",")
	envStr = envStr + "}}"

	return envStr
}

// Function GetGolangEnv returns the golang
// environment variables in a JSON format string.
func GetGolangEnv() string {

	// set environment string
	envStr := fmt.Sprintf("{\"GolangEnvironment\": {\"Version\": \"%s\", \"GOMAXPROCS\": %d, \"NumCPU\": %d, \"GOOS\": \"%s\", \"GOARCH\": \"%s\", \"GOROOT\": \"%s\", \"Compiler\": \"%s\"}}",
		runtime.Version(),
		runtime.GOMAXPROCS(0),
		runtime.NumCPU(),
		strings.ReplaceAll(strings.ReplaceAll(runtime.GOOS, "\\", "/"), ":", "."),
		strings.ReplaceAll(strings.ReplaceAll(runtime.GOARCH, "\\", "/"), ":", "."),
		strings.ReplaceAll(strings.ReplaceAll(runtime.GOROOT(), "\\", "/"), ":", "."),
		runtime.Compiler)

	return envStr
}
