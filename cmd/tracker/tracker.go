package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"sync"

	"github.com/spf13/viper"
	"go-torrent-toy-system/message"
	"go-torrent-toy-system/rpc"
)

const torrentListPath = "/torrents"

//nolint: gochecknoglobals
var (
	activeSeeders = make(map[string][]message.Torrent)
	lock          sync.RWMutex

	handlers = map[message.Type]handlerFuncInterface{
		message.TorrentListType: handleTorrentList,
		message.TorrentType:     handleTorrent,
	}
)

func handleOrReportError(conn net.Conn) {
	errCh := make(chan error, 1)

	go func() {
		if err := handle(conn); err != nil {
			errCh <- err
			return
		}
	}()
	err, ok := <-errCh
	if !ok {
		log.Printf("Tracker: Error occured while handling incoming connection, error: %s", err)
	}
}

func readConfig() (trackerPort int, trackerWebPort int) {
	viper.AddConfigPath("config")
	viper.SetConfigName("config")
	viper.SetConfigType("properties")

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file, %s", err)
	}

	log.Printf("Tracker: Using config: %s\n", viper.ConfigFileUsed())

	trackerPort = viper.GetInt("tracker.port")
	trackerWebPort = viper.GetInt("tracker.webPort")

	return
}

func setUpTracker(trackerPort int, trackerWebPort int) (net.Listener, error) {
	addr := fmt.Sprint("localhost:", trackerPort)
	log.Println("Tracker: address ", addr)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	log.Printf("Tracker: tracker ready on %s", addr)

	webAddr := fmt.Sprint("localhost:", trackerWebPort)

	http.HandleFunc(torrentListPath, webHandler)

	go func() {
		if err := http.ListenAndServe(webAddr, nil); err != nil {
			log.Printf("can't listen on HTTP at %s - %s", webAddr, err)
			return
		}
		log.Printf("Tracker: ready on (web=%s)", webAddr)
	}()

	return ln, nil
}

func webHandler(w http.ResponseWriter, _ *http.Request) {
	lock.RLock()
	defer lock.RUnlock()

	json.NewEncoder(w).Encode(activeSeeders)
}

type handlerFuncInterface func(data []byte, w io.Writer) error

func handleTorrentList(data []byte, _ io.Writer) error {
	var torrentList message.TorrentList
	if err := rpc.UnmarshalPayload(data, &torrentList); err != nil {
		return err
	}

	lock.Lock()
	defer lock.Unlock()
	activeSeeders[torrentList.Seeder] = torrentList.Torrents
	log.Printf("Tracker: added %d torrents list from %+v; torrentList : %v", len(torrentList.Torrents), torrentList.Seeder,
		torrentList.Torrents)

	return nil
}

func handleTorrent(data []byte, w io.Writer) error {
	var torrent message.Torrent
	if err := rpc.UnmarshalPayload(data, &torrent); err != nil {
		return err
	}
	log.Printf("Tracker: received torrent: %+v", torrent)

	seederList := &message.SeederList{
		Torrent: torrent,
		Seeders: findSeeders(torrent),
	}
	log.Printf("Tracker: seeders for torrent %+v: %+v", torrent, seederList)

	data, err := rpc.Marshal(seederList)
	if err != nil {
		return err
	}

	_, err = w.Write(data)
	if err == nil {
		log.Printf("Tracker: wrote back seeders list %v", seederList)
	}

	return err
}

func findSeeders(torrent message.Torrent) []string {
	lock.RLock()
	defer lock.RUnlock()

	var servers []string

	for s, torrentList := range activeSeeders {
		for _, t := range torrentList {
			if torrent == t {
				servers = append(servers, s)
				break
			}
		}
	}

	return servers
}

func handle(conn net.Conn) error {
	remote := conn.RemoteAddr().String()
	log.Printf("Tracker: connected to %s ", remote)

	for {
		typ, data, err := rpc.Decode(conn)
		switch err {
		case nil:
			break
		case io.EOF:
			log.Printf("Tracker: %s: disconnect", remote)
			return nil
		default:
			return fmt.Errorf("no match for %s; error: %v", remote, err)
		}
		log.Printf("Tracker: received message %s with %d bytes", typ, len(data))

		fn, ok := handlers[typ]
		if !ok {
			log.Printf("Tracker: failed to find handler for type %s; skipping", typ)
			continue
		}
		log.Printf("Tracker: handling request of type %s\n", typ)
		if err := fn(data, conn); err != nil {
			return err
		}
	}
}

// torrentList : [{ow.txt %!s(int64=336707) f39631b6df627728ed9d90f6d6e858be5f3061b46a5fdf3a78ce4ec927341692}]
func main() {
	trackerPort, trackerWebPort := readConfig()

	listener, err := setUpTracker(trackerPort, trackerWebPort)
	if err != nil {
		log.Fatalf("Tracker: could not setup listener; %s", err)
		return
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Tracker: can't accept connection - %s", err)
			continue
		}

		handleOrReportError(conn)
	}
}
