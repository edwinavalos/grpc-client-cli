package server

import (
	"context"
	files "github.com/vadimi/grpc-client-cli/gen/go"

	"os"
)

type FileServer struct {
	files.UnimplementedFileServiceServer
}

func NewServer() *FileServer {
	return &FileServer{}
}

func (fs FileServer) ListFiles(ctx context.Context, req *files.ListFilesRequest) (*files.ListFilesResponse, error) {
	f, err := os.ReadDir(".")
	if err != nil {
		return nil, err
	}

	resp := &files.ListFilesResponse{}
	for _, file := range f {
		resp.Files = append(resp.Files, file.Name())
	}
	return resp, nil
}
