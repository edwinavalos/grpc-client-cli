package main

import (
	"context"
	"fmt"
	"github.com/gookit/color"
	"github.com/spf13/viper"
	"github.com/urfave/cli/v2"
	"github.com/vadimi/grpc-client-cli/internal/caller"
	"github.com/vadimi/grpc-client-cli/internal/cliext"
	"github.com/vadimi/grpc-client-cli/internal/rpc"
	"google.golang.org/grpc/status"
	"io"
	"os"
	"runtime/debug"
	"time"

	"github.com/c-bata/go-prompt"
)

type Completer struct {
	servicesList []*caller.ServiceMeta
	opts         *startOpts
	connFact     *rpc.GrpcConnFactory
	endpoints    []Endpoint
	w            io.Writer
}

func NewCompleter(endpoints []Endpoint, startOpts startOpts) *Completer {
	return &Completer{
		opts:      &startOpts,
		endpoints: endpoints,
	}
}

type startOpts struct {
	Service            string
	Method             string
	Discover           bool
	Deadline           int
	Verbose            bool
	Target             string
	IsInteractive      bool
	Authority          string
	InFormat           caller.MsgFormat
	OutFormat          caller.MsgFormat
	OutJsonNames       bool
	GrpcReflectVersion caller.GrpcReflectVersion

	// connection credentials
	TLS      bool
	Insecure bool
	CACert   string
	Cert     string
	CertKey  string

	Protos       []string
	ProtoImports []string
	Headers      map[string][]string

	Keepalive     bool
	KeepaliveTime time.Duration

	MaxRecvMsgSize int
}

func (c *Completer) index(opts *startOpts) (*Completer, error) {
	connOpts := []rpc.ConnFactoryOption{
		rpc.WithAuthority(opts.Authority),
		rpc.WithKeepalive(opts.Keepalive, opts.KeepaliveTime),
	}

	if opts.TLS {
		connOpts = append(connOpts, rpc.WithConnCred(opts.Insecure, opts.CACert, opts.Cert, opts.CertKey))
	}

	if opts.MaxRecvMsgSize > 0 {
		connOpts = append(connOpts, rpc.WithMaxRecvMsgSize(opts.MaxRecvMsgSize))
	}

	if len(opts.Headers) > 0 {
		connOpts = append(connOpts, rpc.WithHeaders(opts.Headers))
	}

	c.connFact = rpc.NewGrpcConnFactory(connOpts...)
	c.opts = opts

	var svc caller.ServiceMetaData
	if len(opts.Protos) > 0 {
		svc = caller.NewServiceMetadataProto(opts.Protos, opts.ProtoImports)
	} else {
		svc = caller.NewServiceMetaData(&caller.ServiceMetaDataConfig{
			ConnFact:       c.connFact,
			Target:         c.opts.Target,
			Deadline:       c.opts.Deadline,
			ProtoImports:   c.opts.ProtoImports,
			ReflectVersion: c.opts.GrpcReflectVersion,
		})
	}

	ctx := rpc.WithStatsCtx(context.Background())
	services, err := svc.GetServiceMetaDataList(ctx)
	if err != nil {
		if c.opts.Verbose {
			printVerbose(c.w, rpc.ExtractRpcStats(ctx), err)
		}
		return nil, err
	}

	additionalFiles, err := svc.GetAdditionalFiles()
	if err != nil {
		return nil, err
	}

	err = caller.RegisterFiles(append(services.Files(), additionalFiles...)...)
	if err != nil && c.opts.Verbose {
		fmt.Println(err)
	}

	c.servicesList = services
	return c, nil
}

func completer(in prompt.Document) []prompt.Suggest {
	s := []prompt.Suggest{}

	return prompt.FilterHasPrefix(s, in.GetWordBeforeCursor(), true)
}

const (
	appVersion = "1.21.1"
)

var helpTemplate = cli.AppHelpTemplate + `
BUILD INFO:
   go version: {{ExtraInfo.go_version}}{{if ExtraInfo.vcs_revision}}
   revision: {{ExtraInfo.vcs_revision}}{{end}}
`

func getExtraInfo() map[string]string {
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return nil
	}

	info := map[string]string{
		"go_version": buildInfo.GoVersion,
	}

	for _, setting := range buildInfo.Settings {
		if setting.Key == "vcs.revision" {
			info["vcs_revision"] = setting.Value
		}
	}

	return info
}

func main() {
	rootLogger := NewLogger()
	viperCfg := viper.New()

	viperCfg.SetConfigName("config")
	viperCfg.AddConfigPath("/etc/tracker")
	viperCfg.AddConfigPath("$HOME/.tracker")
	viperCfg.AddConfigPath(".")
	viperCfg.SetConfigType("yaml")

	err := viperCfg.ReadInConfig()
	if err != nil {
		panic(err)
	}

	cfg := NewConfig()

	err = viperCfg.Unmarshal(&cfg)
	if err != nil {
		panic(err)
	}

	for _, ep := range cfg.Endpoints {
		rootLogger.Infof("Endpoints from config: %s:%d", ep.Address, ep.Port)
	}

	app := cli.NewApp()
	app.Usage = "generic gRPC client"
	app.Version = appVersion
	app.EnableBashCompletion = true
	app.CustomAppHelpTemplate = helpTemplate
	app.ExtraInfo = getExtraInfo

	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:    "service",
			Aliases: []string{"s"},
			Value:   "",
			Usage:   "grpc full or partial service name",
		},
		&cli.StringFlag{
			Name:    "method",
			Aliases: []string{"m"},
			Value:   "",
			Usage:   "grpc service method name",
		},
		&cli.StringFlag{
			Name:    "input",
			Aliases: []string{"i"},
			Value:   "",
			Usage:   "file that contains message json, it will be ignored if used in conjunction with stdin pipes",
		},
		&cli.StringFlag{
			Name:    "deadline",
			Aliases: []string{"d"},
			Value:   "15s",
			Usage:   "grpc call deadline in go duration format, e.g. 15s, 3m, 1h, etc. If no format is specified, defaults to seconds",
		},
		&cli.BoolFlag{
			Name:    "verbose",
			Aliases: []string{"V"},
			Usage:   "output some additional information like request time and message size",
		},
		&cli.BoolFlag{
			Name:  "tls",
			Value: false,
			Usage: "use TLS when connecting to grpc server",
		},
		&cli.BoolFlag{
			Name:  "insecure",
			Value: false,
			Usage: "skip server's certificate chain and host name verification, this option should only be used for testing",
		},
		&cli.StringFlag{
			Name:  "cacert",
			Value: "",
			Usage: "the CA certificate file for verifying the server, this certificate is ignored if --insecure option is true",
		},
		&cli.StringFlag{
			Name:  "cert",
			Value: "",
			Usage: "client certificate to present to the server, only valid with -certkey option",
		},
		&cli.StringFlag{
			Name:  "certkey",
			Value: "",
			Usage: "client private key, only valid with -cert option",
		},
		&cli.StringSliceFlag{
			Name:     "proto",
			Required: false,
			Usage: "proto files or directories to search for proto files, " +
				"if this option is provided service reflection would be ignored. " +
				"In order to provide multiple paths, separate them with comma",
		},
		&cli.StringSliceFlag{
			Name:     "protoimports",
			Required: false,
			Usage:    "additional directories to search for dependencies or supply addtional files for Any type (un)marshal",
		},
		&cli.GenericFlag{
			Name:        "header",
			Aliases:     []string{"H"},
			Required:    false,
			Value:       cliext.NewMapValue(),
			Usage:       "extra header(s) to include in the request",
			DefaultText: "no extra headers",
		},
		&cli.StringFlag{
			Name:  "authority",
			Value: "",
			Usage: "override :authority header",
		},
		&cli.GenericFlag{
			Name:    "informat",
			Aliases: []string{"if"},
			Value: &cliext.EnumValue{
				Enum:    []string{"json", "text"},
				Default: "json",
			},
			Usage: "input proto message format, supported values are json and text",
		},
		&cli.GenericFlag{
			Name:    "outformat",
			Aliases: []string{"of"},
			Value: &cliext.EnumValue{
				Enum:    []string{"json", "text"},
				Default: "json",
			},
			Usage: "output proto message format, supported values are json and text",
		},
		&cli.BoolFlag{
			Name:  "keepalive",
			Value: false,
			Usage: "If true, send keepalive pings even with no active RPCs. If false, default grpc settings are used",
		},
		&cli.DurationFlag{
			Name:        "keepalive-time",
			Usage:       `If set, send keepalive pings every "keepalive-time" timeout. If not set, default grpc settings are used`,
			DefaultText: "not set",
		},
		&cli.IntFlag{
			Name:    "max-receive-message-size",
			Aliases: []string{"mrms", "max-recv-msg-size"},
			Value:   0,
			Usage:   "If greater than 0, sets the max receive message size to bytes, else uses grpc defaults (currently 4 MB)",
		},
		&cli.StringSliceFlag{
			Name:     "endpoint",
			Aliases:  []string{"e", "endpoint"},
			Required: false,
			Usage:    "host:port of the service",
		},
		&cli.BoolFlag{
			Name:  "out-json-names",
			Value: false,
			Usage: "If true uses json_name properties/camel casing in message output",
		},
		&cli.GenericFlag{
			Name: "reflect-version",
			Value: &cliext.EnumValue{
				Enum:    []string{"v1alpha", "auto"},
				Default: "v1alpha",
			},
			Usage: "Specify which grpc reflection version to use, v1alpha is the default as it's the most widely used version for now. " +
				`"auto" option will try to determine the version automatically, it requires correctly functioning grpc server that returns Unimplemented error in case v1 or v1alpha are not supported. ` +
				"After v1 release the default option will be changed.",
		},
	}

	app.Action = baseCmd
	app.Commands = []*cli.Command{
		{
			Name:   "discover",
			Usage:  "print service protobuf",
			Action: discoverCmd,
		},
		{
			Name:   "health",
			Usage:  "grpc health check",
			Action: healthCmd,
		},
	}
	app.Run(os.Args)

	completer := NewCompleter(cfg.Endpoints, sOpts)

	in := prompt.Input(">>> ", completer,
		prompt.OptionTitle("fleetctl"),
		prompt.OptionPrefixTextColor(prompt.Yellow),
		prompt.OptionPreviewSuggestionTextColor(prompt.Blue),
		prompt.OptionSelectedSuggestionBGColor(prompt.LightGray),
		prompt.OptionSuggestionBGColor(prompt.DarkGray))
	fmt.Println("Your input: " + in)

}

func printVerbose(w io.Writer, s *rpc.Stats, rpcErr error) {
	fmt.Fprintln(w)

	fmt.Fprintln(w, color.Bold.Sprint("Method: ")+s.FullMethod())

	rpcStatus := status.Code(rpcErr)
	fmt.Fprintln(w, color.Bold.Sprint("Status: ")+color.FgLightYellow.Sprintf("%d", rpcStatus)+" "+color.OpItalic.Sprint(rpcStatus))

	fmt.Fprintln(w, color.OpItalic.Sprint("\nRequest Headers:"))
	for k, v := range s.ReqHeaders() {
		fmt.Fprintln(w, color.Bold.Sprint(k+": ")+color.LightGreen.Sprint(v))
	}

	if s.RespHeaders().Len() > 0 {
		fmt.Fprintln(w, color.OpItalic.Sprint("\nResponse Headers:"))
		for k, v := range s.RespHeaders() {
			fmt.Fprintln(w, color.Bold.Sprint(k+": ")+color.LightGreen.Sprint(v))
		}
	}

	if s.RespTrailers().Len() > 0 {
		color.Fprintln(w, color.OpItalic.Sprint("\nResponse Trailers:"))
		for k, v := range s.RespTrailers() {
			fmt.Fprintln(w, color.Bold.Sprint(k+": ")+color.LightGreen.Sprint(v))
		}
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, color.Bold.Sprint("Request duration: ")+color.FgLightYellow.Sprint(s.Duration))
	fmt.Fprintln(w, color.Bold.Sprint("Request size: ")+color.FgLightYellow.Sprintf("%d bytes", s.ReqSize()))
	fmt.Fprintln(w, color.Bold.Sprint("Response size: ")+color.FgLightYellow.Sprintf("%d bytes", s.RespSize()))
	fmt.Fprintln(w)
}
