package commander

import (
	"context"
	"flag"
	"fmt"
	go_config "github.com/pefish/go-config"
	"github.com/pefish/go-logger"
	"github.com/pkg/errors"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"
)

type ISubcommand interface {
	DecorateFlagSet(flagSet *flag.FlagSet) error
	Init(data *StartData) error
	Start(data *StartData) error
	OnExited(data *StartData) error
}

type SubcommandInfo struct {
	Desc       string
	Args       []string
	Subcommand ISubcommand
}

type Commander struct {
	subcommands        map[string]*SubcommandInfo
	version            string
	appName            string
	appDesc            string
	fnToSetCommonFlags func(flagSet *flag.FlagSet)

	data *StartData

	cacheFs *os.File

	subCommandValid bool

	cancelFuncOfExitCancelCtx context.CancelFunc
}

type StartData struct {
	DataDir    string
	LogLevel   string
	ConfigFile string
	Cache      Cache

	Args          map[string]string
	ExitCancelCtx context.Context
}

func NewCommander(appName, version, appDesc string) *Commander {
	return &Commander{
		subcommands: make(map[string]*SubcommandInfo),
		version:     version,
		appName:     appName,
		appDesc:     appDesc,
		data: &StartData{
			Args: make(map[string]string, 0),
		},
		subCommandValid: true,
	}
}

func (commander *Commander) RegisterSubcommand(name string, subcommandInfo *SubcommandInfo) {
	commander.subcommands[name] = subcommandInfo
}

// 没有指定子命令的时候，会执行这里注册的子命令
func (commander *Commander) RegisterDefaultSubcommand(subcommandInfo *SubcommandInfo) {
	commander.subcommands["default"] = subcommandInfo
}

// 用于设置所有子命令共用的选项
func (commander *Commander) RegisterFnToSetCommonFlags(flagSet func(set *flag.FlagSet)) {
	commander.fnToSetCommonFlags = flagSet
}

func (commander *Commander) DisableSubCommand() {
	commander.subCommandValid = false
}

func (commander *Commander) hasSubCommand() bool {
	return len(os.Args) != 1 && !strings.HasPrefix(os.Args[1], "-") && commander.subCommandValid
}

func (commander *Commander) Run() error {
	key := "default"
	subcommandInfo := commander.subcommands[key]
	if commander.hasSubCommand() {
		key = os.Args[1]
		subcommandInfo1, ok := commander.subcommands[key]
		if !ok {
			return errors.Errorf("subcommand <%s> not found!", key)
		}
		subcommandInfo = subcommandInfo1
	}

	flagSet := flag.NewFlagSet(commander.appName, flag.ExitOnError)
	flagSet.Usage = func() {

		fmt.Printf("\n%s\n\n", commander.appDesc)

		fmt.Printf("Usage of <%s>:\n", flagSet.Name())
		usageStr := fmt.Sprintf("  %s", flagSet.Name())
		if commander.subCommandValid {
			usageStr += " [subcommand]"
		}
		usageStr += " [options]"
		if !commander.subCommandValid && len(commander.subcommands["default"].Args) > 0 {
			for _, arg := range commander.subcommands["default"].Args {
				usageStr += fmt.Sprintf(" <%s>", arg)
			}
		}
		fmt.Printf("%s\n\n", usageStr)

		// 如果有子命令，打印所有子命令
		if commander.subCommandValid {
			if len(commander.subcommands) > 0 {
				fmt.Println("Commands:")
				for name, info := range commander.subcommands {
					argsStrArr := make([]string, len(info.Args))
					for i, arg := range info.Args {
						argsStrArr[i] = fmt.Sprintf("<%s>", arg)
					}
					argsStr := strings.Join(argsStrArr, " ")
					if name == "default" {
						fmt.Printf("  [default] %s\tDefault subcommand. %s\n", argsStr, info.Desc)
					} else {
						fmt.Printf("  %s %s\t%s\n", name, argsStr, info.Desc)
					}
				}
				fmt.Printf("\n")
			}
		}

		fmt.Println("Options:")
		flagSet.PrintDefaults()
		fmt.Printf("\n")
	}
	flagSet.Bool("version", false, "print version string")
	flagSet.String("log-level", "info", "set log verbosity: debug, info, warn, or error")
	configFile := flagSet.String("config", os.Getenv("GO_CONFIG"), "path to config file")
	flagSet.Bool("enable-pprof", false, "enable pprof")
	flagSet.String("pprof-address", "0.0.0.0:9191", "<addr>:<port> to listen on for pprof")
	flagSet.String("data-dir", os.ExpandEnv("$HOME/.")+commander.appName, "data dictionary")
	if commander.fnToSetCommonFlags != nil {
		commander.fnToSetCommonFlags(flagSet)
	}
	if subcommandInfo != nil {
		err := subcommandInfo.Subcommand.DecorateFlagSet(flagSet)
		if err != nil {
			return errors.Wrap(err, "decorate flagSet error")
		}
	}
	argsToParse := os.Args[1:]
	if commander.hasSubCommand() {
		argsToParse = os.Args[2:]
	}
	err := flagSet.Parse(argsToParse)
	if err != nil {
		return errors.Wrap(err, "parse flagSet error")
	}

	// merge envs and config file
	err = go_config.ConfigManagerInstance.LoadConfig(go_config.Configuration{
		ConfigFilepath: *configFile,
	})
	if err != nil {
		return errors.Errorf("load config file error - %s", err)
	}
	go_config.ConfigManagerInstance.MergeFlagSet(flagSet)
	envKeyPairs := make(map[string]string, 5)
	for k, _ := range go_config.ConfigManagerInstance.Configs() {
		env := strings.ReplaceAll(strings.ToUpper(k), "-", "_")
		envKeyPairs[env] = k
	}
	go_config.ConfigManagerInstance.MergeEnvs(envKeyPairs)

	logLevel, err := go_config.ConfigManagerInstance.GetString("log-level")
	if err != nil {
		return errors.Wrap(err, "get log-level config error")
	}
	commander.data.LogLevel = logLevel
	go_logger.Logger = go_logger.NewLogger(logLevel)

	commander.data.ConfigFile = *configFile
	for i, arg := range subcommandInfo.Args {
		if i > len(flagSet.Args())-1 {
			return errors.Errorf("arg <%s> not be set", arg)
		}
		commander.data.Args[arg] = flagSet.Args()[i]
	}
	ctx, cancel := context.WithCancel(context.Background())
	commander.data.ExitCancelCtx = ctx
	commander.cancelFuncOfExitCancelCtx = cancel

	dataDirStr, err := go_config.ConfigManagerInstance.GetString("data-dir")
	if err != nil {
		return errors.Wrap(err, "get data-dir config error")
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
	commander.data.DataDir = dataDirStr

	printVersion, err := go_config.ConfigManagerInstance.GetBool("version")
	if err != nil {
		return errors.Wrap(err, "get version config error")
	}
	if printVersion {
		fmt.Println(commander.version)
		os.Exit(0)
	}

	if subcommandInfo == nil {
		return errors.Errorf("subcommand error: %s subcommand not found", key)
	}

	pprofEnable, err := go_config.ConfigManagerInstance.GetBool("enable-pprof")
	if err != nil {
		return errors.Wrap(err, "get enable-pprof config error")
	}
	pprofAddress, err := go_config.ConfigManagerInstance.GetString("pprof-address")
	if err != nil {
		return errors.Wrap(err, "get version config error")
	}
	if pprofEnable {
		pprofHttpServer := &http.Server{Addr: pprofAddress}
		go func() { // 无需担心进程退出，不存在leak
			go_logger.Logger.InfoF("started pprof server on %s, you can open url [http://%s/debug/pprof/] to enjoy!!", pprofHttpServer.Addr, pprofHttpServer.Addr)
			err := pprofHttpServer.ListenAndServe()
			if err != nil {
				go_logger.Logger.WarnF("pprof server start error - %s", err)
			}
		}()
	}

	// load cache
	err = commander.data.Cache.Init(path.Join(commander.data.DataDir, "data.json"))
	if err != nil {
		return err
	}

	err = subcommandInfo.Subcommand.Init(commander.data)
	if err != nil {
		return err
	}

	waitErrorChan := make(chan error)
	go func() {
		waitErrorChan <- subcommandInfo.Subcommand.Start(commander.data)
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
				commander.cancelFuncOfExitCancelCtx() // 通知下去，程序即将退出
				go_logger.Logger.Info("Got interrupt, exiting...")
			} else {
				go_logger.Logger.InfoF("Got interrupt, exiting... %d", ctrlCCountTemp)
			}
			ctrlCCountTemp--
			if ctrlCCountTemp <= 0 { // Ctrl C n 次强制退出，不等 start 函数了
				break forceExit
			}
			break
		case exitErr = <-waitErrorChan:
			break forceExit
		}
	}

	err = commander.onExitedBefore()
	if err != nil {
		exitErr = errors.WithMessage(exitErr, fmt.Sprintf("commander OnExitedBefore failed - %s", err.Error()))
	}
	err = subcommandInfo.Subcommand.OnExited(commander.data)
	if err != nil {
		exitErr = errors.WithMessage(exitErr, fmt.Sprintf("OnExited failed - %s", err.Error()))
	}
	err = commander.onExitedAfter()
	if err != nil {
		exitErr = errors.WithMessage(exitErr, fmt.Sprintf("commander OnExitedAfter failed - %s", err.Error()))
	}
	return exitErr
}

func (commander *Commander) onExitedAfter() error {
	return nil
}

func (commander *Commander) onExitedBefore() error {
	return nil
}
