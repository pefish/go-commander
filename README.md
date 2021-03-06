# go-commander

[![view examples](https://img.shields.io/badge/learn%20by-examples-0C8EC5.svg?style=for-the-badge&logo=go)](https://github.com/pefish/go-commander)

go-commander

## Quick start

```go
package main

import (
	"flag"
	"fmt"
	commander2 "github.com/pefish/go-commander"
	go_config "github.com/pefish/go-config"
	"log"
)


type TestSubCommand struct {

}

func (t TestSubCommand) DecorateFlagSet(flagSet *flag.FlagSet) error {
	flagSet.String("test", "haha", "")
	return nil
}

func (t TestSubCommand) Start() error {
	fmt.Println(go_config.Config.MustGetString("test"))
	fmt.Println(go_config.Config.MustGetString("testtest"))
	return nil
}

func main() {
	commander := commander2.NewCommander("test", "v0.0.1", "")
	commander.RegisterSubcommand("test", TestSubCommand{})
	commander.RegisterDefaultSubcommand(TestSubCommand{})
	commander.RegisterFnToSetCommonFlags(func(flagSet *flag.FlagSet) {
		flagSet.String("testtest", "", "path to config file")
	})
	err := commander.Run()
	if err != nil {
		log.Fatal(err)
	}
}

```

```shell script
go run ./_example/main.go -test=76573 -testtest=11
```

## Document

[doc](https://godoc.org/github.com/pefish/go-commander)

默认包含的选项

* version。打印版本信息
* log-level。设置日志打印级别
* config。设置配置文件的位置，默认值是GO_CONFIG环境变量。（命令行选项 > 环境变量 > 配置文件）
* secret-file。设置secret配置文件的位置，默认值是GO_SECRET环境变量。
* enable-pprof。是否启用pprof分析工具
* pprof-address。pprof分析工具的监听地址

支持3中配置输入：

1. 命令行选项
2. 环境变量
3. 配置文件

优先级是：命令行选项 > 环境变量 > 配置文件 > 默认值

**注意：环境变量传递配置项时，`-`使用`_`代替，例如`TEST_TEST`将会代替`test-test`，也会代替`test_test`**

## Security Vulnerabilities

If you discover a security vulnerability, please send an e-mail to [pefish@qq.com](mailto:pefish@qq.com). All security vulnerabilities will be promptly addressed.

## License

This project is licensed under the [Apache License](LICENSE).
