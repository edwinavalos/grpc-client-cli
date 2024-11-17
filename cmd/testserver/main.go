package main

import (
	"fmt"
	"github.com/spf13/viper"
	"github.com/vadimi/grpc-client-cli/cmd/testserver/server"
	files "github.com/vadimi/grpc-client-cli/gen/go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"net"
	"os"
	"os/signal"
	"syscall"
)

var rootLogger *Logger

func serve(svrCfg Endpoint) {
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", svrCfg.Address, svrCfg.Port))
	if err != nil {
		rootLogger.Errorf("failed to listen: %v", err)
		os.Exit(1)
	}

	fs := server.NewServer()
	grpcServer := grpc.NewServer()
	files.RegisterFileServiceServer(grpcServer, fs)
	reflection.Register(grpcServer)

	rootLogger.Infof("Starting server on port %s:%d", svrCfg.Address, svrCfg.Port)
	err = grpcServer.Serve(lis)
	if err != nil {
		panic(err)
	}
}

func main() {
	rootLogger = NewLogger()
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

	for _, svrCfg := range cfg.Endpoints {
		go serve(svrCfg)
	}

	sigs := make(chan os.Signal, 1)

	// Register the signals you want to handle.
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// Block until a signal is received.
	fmt.Println("Waiting for signal...")
	<-sigs
	fmt.Println("Signal received, exiting...")

}
