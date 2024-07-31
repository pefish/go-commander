package commander

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path"
	"reflect"
	"slices"
	"strings"
	"syscall"

	go_config "github.com/pefish/go-config"
	go_file "github.com/pefish/go-file"
	go_format "github.com/pefish/go-format"
	i_logger "github.com/pefish/go-interface/i-logger"
	t_logger "github.com/pefish/go-interface/t-logger"
	go_logger "github.com/pefish/go-logger"
	"github.com/pkg/errors"
)

type ISubcommand interface {
	Config() interface{}
	Data() interface{} // 应用数据。应用启动时自动从应用目录加载数据，应用退出是自动保存到应用目录
	Init(data *Commander) error
	Start(data *Commander) error
	OnExited(data *Commander) error
}

type BasicConfig struct {
	Version      bool   `json:"version"`
	LogLevel     string `json:"log-level"`
	Config       string `json:"config"`
	EnablePprof  bool   `json:"enable-pprof"`
	PprofAddress string `json:"pprof-address"`
	DataDir      string `json:"data-dir"`
}

type SubcommandInfo struct {
	Desc       string
	Args       []string
	Subcommand ISubcommand
}

type Commander struct {
	subcommands map[string]*SubcommandInfo
	version     string
	appName     string
	appDesc     string
	cache       Cache

	Name       string
	DataDir    string
	LogLevel   string
	ConfigFile string
	Args       map[string]string
	Ctx        context.Context
	CancelFunc context.CancelFunc
	Logger     i_logger.ILogger
}

func NewCommander(appName, version, appDesc string) *Commander {
	return &Commander{
		subcommands: make(map[string]*SubcommandInfo),
		version:     version,
		appName:     appName,
		appDesc:     appDesc,
		Args:        make(map[string]string),
	}
}

func (commander *Commander) RegisterSubcommand(name string, subcommandInfo *SubcommandInfo) {
	commander.subcommands[name] = subcommandInfo
}

// 没有指定子命令的时候，会执行这里注册的子命令
func (commander *Commander) RegisterDefaultSubcommand(subcommandInfo *SubcommandInfo) {
	commander.subcommands["default"] = subcommandInfo
}

func (commander *Commander) Run() error {
	flagSet := flag.NewFlagSet(commander.appName, flag.ExitOnError)
	flagSet.Bool("version", false, "print version string")
	flagSet.String("log-level", "info", "set log verbosity: debug, info, warn, or error")
	configFile := flagSet.String("config", os.Getenv("GO_CONFIG"), "path to config file")
	envFile := flagSet.String("env-file", ".env", "path to env file")
	flagSet.Bool("enable-pprof", false, "enable pprof")
	flagSet.String("pprof-address", "0.0.0.0:9191", "<addr>:<port> to listen on for pprof")
	flagSet.String("data-dir", os.ExpandEnv("$HOME/.")+commander.appName, "data dictionary")

	var subcommandInfo *SubcommandInfo
	argsToParse := os.Args[1:]
	if len(argsToParse) > 0 && !strings.HasPrefix(argsToParse[0], "-") {
		// 输入了子命令
		commander.Name = os.Args[1]
		argsToParse = os.Args[2:]

		subcommandInfo_, ok := commander.subcommands[commander.Name]
		if !ok {
			fmt.Printf("%s: '%s' is not a command.\n", commander.appName, commander.Name)
			fmt.Printf("See '%s --help'\n", commander.appName)
			return nil
		}
		subcommandInfo = subcommandInfo_

		flagSetJustForPrintHelpInfo := flag.NewFlagSet(commander.appName, flag.ExitOnError)
		if subcommandInfo.Subcommand.Config() != nil {
			// 将传进来的 Config 对象加载到 flagSet 中，使其能正常打印帮助信息
			err := parseConfigToFlagSet(flagSetJustForPrintHelpInfo, subcommandInfo.Subcommand.Config())
			if err != nil {
				return errors.Wrap(err, "ParseConfigToFlagSet flagSet error.")
			}
		}

		flagSet.Usage = func() {
			argsStrArr := make([]string, len(subcommandInfo.Args))
			for i, arg := range subcommandInfo.Args {
				argsStrArr[i] = fmt.Sprintf("<%s>", arg)
			}
			fmt.Printf(
				`
%s

Usage:
  %s %s [OPTIONS] -- %s

Options:
`,
				subcommandInfo.Desc,
				commander.appName,
				commander.Name,
				strings.Join(argsStrArr, " "),
			)
			flagSetJustForPrintHelpInfo.PrintDefaults()
			fmt.Printf(`
Global Options:
`)
			flagSet.PrintDefaults()
			fmt.Printf("\n")
		}
	} else {
		flagSet.Usage = func() {

			fmt.Printf("\n%s\n\n", commander.appDesc)

			fmt.Printf("Usage:\n")
			usageStr := fmt.Sprintf("  %s", flagSet.Name())
			if len(commander.subcommands) > 0 {
				usageStr += " [COMMAND]"
			}
			usageStr += " [OPTIONS]"
			fmt.Printf("%s\n\n", usageStr)

			// 如果有子命令，打印所有子命令
			if len(commander.subcommands) > 0 {
				fmt.Println("SubCommands:")
				for name, info := range commander.subcommands {
					argsStrArr := make([]string, len(info.Args))
					for i, arg := range info.Args {
						argsStrArr[i] = fmt.Sprintf("<%s>", arg)
					}
					argsStr := ""
					if len(argsStrArr) > 0 {
						argsStr = fmt.Sprintf(" -- %s", strings.Join(argsStrArr, " "))
					}
					if name == "default" {
						fmt.Printf("  %s [OPTIONS]%s\t%s\n", commander.appName, argsStr, info.Desc)
					} else {
						fmt.Printf("  %s %s [OPTIONS]%s\t%s\n", commander.appName, name, argsStr, info.Desc)
					}
				}
				fmt.Printf("\n")
			}

			fmt.Println("Global Options:")
			flagSet.PrintDefaults()
			fmt.Printf("\n")
		}

		commander.Name = "default"
		subcommandInfo_, ok := commander.subcommands[commander.Name]
		// default 命令也没有注册
		if !ok {
			flagSet.Usage()
			return nil
		}
		subcommandInfo = subcommandInfo_
	}

	if slices.Contains(argsToParse, "--help") || slices.Contains(argsToParse, "-help") {
		flagSet.Usage()
		return nil
	}

	// 自定义参数
	err := parseConfigToFlagSet(flagSet, subcommandInfo.Subcommand.Config())
	if err != nil {
		return errors.Wrap(err, "ParseConfigToFlagSet flagSet error.")
	}

	err = flagSet.Parse(argsToParse)
	if err != nil {
		return errors.Wrap(err, "Parse flagSet error.")
	}

	if go_file.FileInstance.Exists(*envFile) {
		err = go_config.ConfigManagerInstance.SetEnvFile(*envFile)
		if err != nil {
			return errors.Errorf("Set env file error - %s", err)
		}
	}
	go_config.ConfigManagerInstance.MergeFlagSet(flagSet)
	if configFile != nil && *configFile != "" {
		err = go_config.ConfigManagerInstance.MergeConfigFile(*configFile)
		if err != nil {
			return errors.Errorf("Load config file error - %s", err)
		}
		commander.ConfigFile = *configFile
	}
	if subcommandInfo.Subcommand.Config() != nil {
		err := go_config.ConfigManagerInstance.Unmarshal(subcommandInfo.Subcommand.Config())
		if err != nil {
			return errors.Errorf("Unmarshal config error - %s", err)
		}
	}

	logLevel, err := go_config.ConfigManagerInstance.String("log-level")
	if err != nil {
		return errors.Wrap(err, "Get log-level config error.")
	}
	commander.LogLevel = logLevel
	commander.Logger = go_logger.NewLogger(t_logger.Level(logLevel))

	args := make([]string, 0)
	argsStartIndex := len(os.Args) - 1
	for i, a := range os.Args {
		if a == "--" {
			argsStartIndex = i
			continue
		}
		if i > argsStartIndex {
			args = append(args, a)
		}
	}
	for i, arg := range subcommandInfo.Args {
		if i > len(args)-1 {
			return errors.Errorf("Arg <%s> not be set. args: %#v", arg, args)
		}
		commander.Args[arg] = args[i]
	}
	ctx, cancel := context.WithCancel(context.Background())
	commander.Ctx = ctx
	commander.CancelFunc = cancel

	dataDirStr, err := go_config.ConfigManagerInstance.String("data-dir")
	if err != nil {
		return errors.Wrap(err, "Get data-dir config error.")
	}
	fsStat, err := os.Stat(dataDirStr)
	if err != nil {
		if strings.Contains(err.Error(), "no such file or directory") || fsStat == nil || !fsStat.IsDir() {
			err = os.Mkdir(dataDirStr, 0755)
			if err != nil {
				return err
			}
			go_logger.Logger.DebugF("%s created", dataDirStr)
		} else {
			return err
		}
	}
	commander.DataDir = dataDirStr

	printVersion, err := go_config.ConfigManagerInstance.Bool("version")
	if err != nil {
		return errors.Wrap(err, "Get version config error.")
	}
	if printVersion {
		fmt.Println(commander.version)
		os.Exit(0)
	}

	pprofEnable, err := go_config.ConfigManagerInstance.Bool("enable-pprof")
	if err != nil {
		return errors.Wrap(err, "Get enable-pprof config error.")
	}
	pprofAddress, err := go_config.ConfigManagerInstance.String("pprof-address")
	if err != nil {
		return errors.Wrap(err, "Get version config error.")
	}
	if pprofEnable {
		pprofHttpServer := &http.Server{Addr: pprofAddress}
		go func() { // 无需担心进程退出，不存在 leak
			go_logger.Logger.InfoF("Started pprof server on %s, you can open url [http://%s/debug/pprof/] to enjoy!!", pprofHttpServer.Addr, pprofHttpServer.Addr)
			err := pprofHttpServer.ListenAndServe()
			if err != nil {
				go_logger.Logger.WarnF("Pprof server start error - %s", err)
			}
		}()
	}

	// load cache
	err = commander.cache.Init(path.Join(commander.DataDir, "data.json"))
	if err != nil {
		return err
	}

	if subcommandInfo.Subcommand.Data() != nil {
		_, err = commander.cache.Load(subcommandInfo.Subcommand.Data())
		if err != nil {
			return err
		}
	}

	err = subcommandInfo.Subcommand.Init(commander)
	if err != nil {
		return err
	}

	waitErrorChan := make(chan error)
	go func() {
		waitErrorChan <- subcommandInfo.Subcommand.Start(commander)
	}()

	exitChan := make(chan os.Signal)
	signal.Notify(exitChan, syscall.SIGINT, syscall.SIGTERM)

	var exitErr error

	ctrlCCount := 3
	ctrlCCountTemp := ctrlCCount
forceExit:
	for {
		select {
		case <-exitChan:
			// 要等待 start 函数退出
			if ctrlCCountTemp == ctrlCCount {
				commander.CancelFunc() // 通知下去，程序即将退出
				go_logger.Logger.Info("Got interrupt, exiting...")
			} else {
				go_logger.Logger.InfoF("Got interrupt, exiting... %d", ctrlCCountTemp)
			}
			ctrlCCountTemp--
			if ctrlCCountTemp <= 0 { // Ctrl C n 次强制退出，不等 start 函数了
				break forceExit
			}
		case exitErr = <-waitErrorChan:
			break forceExit
		}
	}

	err = commander.onExitedBefore()
	if err != nil {
		exitErr = errors.WithMessage(exitErr, fmt.Sprintf("Commander OnExitedBefore failed - %s", err.Error()))
	}
	if subcommandInfo.Subcommand.Data() != nil {
		err = commander.cache.Save(subcommandInfo.Subcommand.Data())
		if err != nil {
			return err
		}
	}
	err = subcommandInfo.Subcommand.OnExited(commander)
	if err != nil {
		exitErr = errors.WithMessage(exitErr, fmt.Sprintf("OnExited failed - %s", err.Error()))
	}
	err = commander.onExitedAfter()
	if err != nil {
		exitErr = errors.WithMessage(exitErr, fmt.Sprintf("Commander OnExitedAfter failed - %s", err.Error()))
	}
	return exitErr
}

func (commander *Commander) onExitedAfter() error {
	return nil
}

func (commander *Commander) onExitedBefore() error {
	return nil
}

// parseConfigToFlagSet dynamically parses the config struct and sets up flags
func parseConfigToFlagSet(flagSet *flag.FlagSet, config interface{}) error {
	// Get the type and value of the config
	configType := reflect.TypeOf(config)
	configValue := reflect.ValueOf(config)

	// If the config is a pointer, get the element it points to
	if configType.Kind() == reflect.Ptr {
		configType = configType.Elem()
		configValue = configValue.Elem()
	}

	// Iterate over the fields of the config struct
	for i := 0; i < configType.NumField(); i++ {
		field := configType.Field(i)
		fieldValue := configValue.Field(i)

		// Get the tag values
		jsonTag := field.Tag.Get("json")
		defaultTag := field.Tag.Get("default")
		usageTag := field.Tag.Get("usage")

		// Use the tag values to define the flags
		switch fieldValue.Kind() {
		case reflect.String:
			flagSet.String(jsonTag, defaultTag, usageTag)
		case reflect.Bool:
			defaultValue := false
			if defaultTag == "true" {
				defaultValue = true
			}
			flagSet.Bool(jsonTag, defaultValue, usageTag)
		case reflect.Int:
			defaultValue := 0
			if defaultTag != "" {
				i, err := go_format.FormatInstance.ToInt(defaultTag)
				if err != nil {
					return errors.Wrapf(err, "Default tag <%s> to int error.", defaultTag)
				}
				defaultValue = i
			}
			flagSet.Int(jsonTag, defaultValue, usageTag)
		case reflect.Float64:
			defaultValue := 0.0
			if defaultTag != "" {
				i, err := go_format.FormatInstance.ToFloat64(defaultTag)
				if err != nil {
					return errors.Wrapf(err, "Default tag <%s> to int error.", defaultTag)
				}
				defaultValue = i
			}
			flagSet.Float64(jsonTag, defaultValue, usageTag)
		default:
			if strings.Contains(fieldValue.String(), "commander.BasicConfig") {
				continue
			}
			return errors.Errorf("Type <%s> not be supported.", fieldValue.Kind().String())
		}
	}

	return nil
}
