package main

import (
	"context"
	"fmt"
	"go-torrent-toy-system/util"
	"log"
	"net"
	"os"
	"time"

	"github.com/spf13/viper"
	"go-torrent-toy-system/files"
	"go-torrent-toy-system/message"
	"go-torrent-toy-system/rpc"
)

const numberOfSeconds = 10

func readConfig() (outputFileName string, chunkSize int64, tor *message.Torrent, trackerPort int) {
	viper.AddConfigPath("config")
	viper.SetConfigName("config")
	viper.SetConfigType("properties")

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file, %s", err)
	}

	log.Printf("Using config: %s\n", viper.ConfigFileUsed())

	outputFileName = viper.GetString("client.outputFileName")
	chunkSize = viper.GetInt64("client.chunkSize")
	tor = &message.Torrent{
		Name: viper.GetString("client.fileName"),
		Size: viper.GetInt64("client.fileSize"),
		Hash: viper.GetString("client.fileHash"),
	}

	trackerPort = viper.GetInt("tracker.port")

	return
}

func main() {
	outputFileName, chunkSize, tor, trackerPort := readConfig()

	sl, err := findSeeders(fmt.Sprintf("localhost:%d", trackerPort), tor)
	if err != nil {
		log.Fatal("Client : ", err)
		return
	}

	if len(sl.Seeders) == 0 {
		log.Fatalf("Client: no seeders for %+#v", tor)
		return
	}

	log.Printf("Client: %d seeders: %s", len(sl.Seeders), sl.Seeders)

	if err := files.CreateEmptyFile(outputFileName, tor.Size); err != nil {
		log.Fatal("Client: failed to create an empty file, error: ", err)
		return
	}

	reqs := splitToChunks(tor.Name, tor.Size, chunkSize)
	log.Printf("Client: using %d workers, workder are %v", len(reqs), reqs)
	out := make(chan *jobResponse)

	ctx, cancel := context.WithTimeout(context.Background(), numberOfSeconds*time.Second)

	for i, req := range reqs {
		addr := sl.Seeders[i%len(sl.Seeders)]
		go chunkWorker(ctx, addr, req, out, outputFileName)
	}

	ok := true

	for range reqs {
		jr := <-out
		if jr.err != nil {
			log.Printf("Client: error reading  %s", jr.err)
			cancel()
			ok = false
		}
	}

	if !ok {
		os.Exit(1)
	}

	log.Printf("Client: download finished")

	size, hash, err := files.FileInfo(outputFileName)
	if err != nil {
		log.Fatal(err)
	}

	if size != tor.Size || hash != tor.Hash {
		log.Fatal("Client: downloaded bad file")
	}

	log.Printf("Client: %s downloaded", outputFileName)
}

//nolint: interfacer
func findSeeders(addr string, torrent *message.Torrent) (*message.SeederList, error) {
	var conn, err = net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}

	if conn != nil {
		defer util.Close(conn)
	}

	var sl message.SeederList
	if err := rpc.Call(conn, torrent, &sl); err != nil {
		return nil, err
	}

	return &sl, nil
}

func min(a, b int64) int64 {
	if a < b {
		return a
	}

	return b
}

func splitToChunks(name string, size int64, chunkSize int64) []*message.ChunkRequest {
	var reqs []*message.ChunkRequest

	for offset := int64(0); offset < size; offset += chunkSize {
		r := &message.ChunkRequest{
			FileName: name,
			Offset:   offset,
			Size:     min(size-offset, chunkSize),
		}
		reqs = append(reqs, r)
	}

	return reqs
}

func chunkWorker(ctx context.Context, addr string, req *message.ChunkRequest,
	out chan<- *jobResponse, outputDir string) {
	jr := &jobResponse{
		req: req,
		err: nil,
	}

	defer func() { out <- jr }()

	var d net.Dialer

	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		jr.err = err
		return
	}

	var resp message.ChunkResponse
	if err := rpc.Call(conn, req, &resp); err != nil {
		jr.err = err
		return
	}

	if err := files.WriteAt(outputDir, req.Offset, resp.Data); err != nil {
		jr.err = err
		return
	}
}

type jobResponse struct {
	req *message.ChunkRequest
	err error
}
