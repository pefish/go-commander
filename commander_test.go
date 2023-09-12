package commander

import (
	"fmt"
	"os/exec"
)

func ExampleCommander() {
	cmd := exec.Command("/bin/bash", "-c", "go run ./_example/main.go -test=cmd-test")
	result, err := cmd.Output()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(result))

	cmd1 := exec.Command("/bin/bash", "-c", "go run ./_example/main.go -test=cmd-test abc")
	result1, err := cmd1.Output()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(result1))

	cmd2 := exec.Command("/bin/bash", "-c", " go run ./_example/main.go -config=./_example/config.yaml abc")
	result2, err := cmd2.Output()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(result2))

	cmd3 := exec.Command("/bin/bash", "-c", "TEST=env-test go run ./_example/main.go default abc")
	result3, err := cmd3.Output()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(result3))

	// Output:
	// ERROR	arg <file> not be set
	//
	// INFO	test: cmd-test, args: map[string]string{"file":"abc"}
	//
	// INFO	test: config-file-test, args: map[string]string{"file":"abc"}
	//
	// INFO	test: env-test, args: map[string]string{"file":"abc"}
}
