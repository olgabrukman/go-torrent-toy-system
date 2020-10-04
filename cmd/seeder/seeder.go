package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"path/filepath"

	//nolint:goimports
	"github.com/spf13/viper"
	"go-torrent-toy-system/files"
	"go-torrent-toy-system/message"
	"go-torrent-toy-system/rpc"
	"go-torrent-toy-system/util"
)

func readConfig() (inputDirName string, seederPort int, trackerPort int) {
	viper.AddConfigPath("config")
	viper.SetConfigName("config")
	viper.SetConfigType("properties")

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file, %s", err)
	}
	log.Printf("Seeder: Using config: %s\n", viper.ConfigFileUsed())

	inputDirName = viper.GetString("seeder.inputDir")
	seederPort = viper.GetInt("seeder.port")
	trackerPort = viper.GetInt("tracker.port")
	return
}

func Close(c io.Closer) {
	err := c.Close()
	if err != nil {
		log.Fatal(err)
	}
}

func reportError(ch chan error) {
	err, ok := <-ch
	if ok {
		log.Printf("Seeder: Error occurred while handling connection, error: %s", err)
	}
}

func handle(conn net.Conn, inputDirName string) error {
	if conn != nil {
		defer util.Close(conn)
	}
	remote := conn.RemoteAddr()
	log.Printf("Seeder: client [%s] connected", remote)

	for {
		typ, data, err := rpc.Decode(conn)
		if err != nil {
			return err
		}

		log.Printf("Seeder: client %s sent request of type %v ", remote, typ)

		if typ != message.ChunkRequestType {
			log.Printf("Seeder: [%s] ignore %s request", remote, typ)
			continue
		}

		var req message.ChunkRequest
		if err := rpc.UnmarshalPayload(data, &req); err != nil {
			log.Printf("Seeder: [%s] error decoding payload - %s", remote, err)
			continue
		}
		log.Printf("Seeder: client's request is %v ", req)

		resp := fillRequest(&req, inputDirName)
		log.Printf("Seeder: filled in response, size: %d, text: %s\n", resp.Size, resp.Data)

		data, err = rpc.Marshal(resp)
		if err != nil {
			log.Printf("Seeder: [%s] error encoding response - %s", remote, err)
			continue
		}

		n, err := conn.Write(data)

		log.Println("Sent response to client")

		if err != nil || n == 0 {
			log.Printf("Seeder: [%s] error writing response %s,  error: %v", remote, data, err)
		}
	}
}

func fillRequest(r *message.ChunkRequest, rootDir string) *message.ChunkResponse {
	resp := &message.ChunkResponse{
		ChunkRequest: *r,
	}

	buf := make([]byte, r.Size)
	fullFileName := fmt.Sprint(rootDir, "/", r.FileName)
	log.Println("Seeder: reading chunk from file ", fullFileName)

	if err := files.ReadAt(fullFileName, r.Offset, buf); err != nil {
		resp.Error = err.Error()
		return resp
	}

	resp.Data = buf
	return resp
}

func generateTorrentsFromFiles(rootDir string) ([]message.Torrent, error) {
	matches, err := filepath.Glob(fmt.Sprintf("%s/*", rootDir))
	if err != nil {
		return nil, err
	}

	//nolint:prealloc
	var torrents []message.Torrent

	for _, name := range matches {
		if !files.IsFile(name) {
			continue
		}

		size, hash, err := files.FileInfo(name)
		if err != nil {
			return nil, err
		}
		t := message.Torrent{
			Name: name[len(rootDir)+1:],
			Size: size,
			Hash: hash,
		}
		torrents = append(torrents, t)
	}

	return torrents, nil
}

func generateTorrentsAndUpdateTracker(addr string, seederAddr string, rootDir string) error {
	torrents, err := generateTorrentsFromFiles(rootDir)
	if err != nil {
		return err
	}
	log.Println("List of torrents", torrents)

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}

	defer util.Close(conn)

	msg := &message.TorrentList{
		Seeder:   seederAddr,
		Torrents: torrents,
	}

	return rpc.Call(conn, msg, nil)
}

func main() {
	inputDirName, seederPort, trackerPort := readConfig()

	addr := fmt.Sprintf("localhost:%d", seederPort)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Seeder: can't setup listener; error %s", err)
	}
	defer Close(listener)

	if err := generateTorrentsAndUpdateTracker(fmt.Sprintf("localhost:%d", trackerPort), addr, inputDirName); err != nil {
		log.Fatalf("Seeder: failed to update tracker; %s", err)
	}
	log.Printf("Seeder: updated tracker, ready on %s", addr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatalf("Seeder: can't accept connections; error: %s", err)
		}
		errCh := make(chan error, 1)

		go func() {
			if err := handle(conn, inputDirName); err != nil {
				errCh <- err
				return
			}
		}()
		reportError(errCh)
	}
}
