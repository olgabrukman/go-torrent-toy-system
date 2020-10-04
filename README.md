## Toy project in GoLang. 

The system is a simplified bit torrent system. There are three main components:
1. Client 
2. Tracker
3. Seeder

### Tracker
The tracker holds a list of active seeders and torrents the seeders they hold.
* Receive list of torrents from seeder and update internal database
* Return list of seeders holding a torrent
* Give it a web API to query the list of active seeders & how many torrents each one holds

### Client
A client sends a torrent request to a tracker.
* Query tracker for a torrent and download it from list of seeders
* Support timeout in download using context.Context
* Verify the downloaded file matches signature in torrent

### Seeder 
The seeder sends its identifiers (host, port) and list of torrents it hosts to the tracker 
on start up. 

In this system, the seeder seeds file "data/aow.txt", with size 343691 bytes and
sha256 hash code 98c4a4a4710a40ae79de349b582950f9eed2196a7ff05f1e4c5aca3b51d1f588.
The client ask to get ```config/config.properties/client.fileName``` file of size 
```config/config.properties/client.fileSize``` and 
```config/config.properties/client.fileHash``` hash code. The result will be stored 
in file ```config/config.properties/client.outputFileName```.

## GoLang Tips
Replace ```go handle(conn)```, where handle may return error with the following code 
to handle a possible error:
```
func handleOrReportError(conn net.Conn) {
   	errCh := make(chan error, 1)
   	go func() {
   		if err := handle(conn); err != nil {
   			errCh <- err
   			return
   		}
   	}()
   	err, ok := <-errCh
   	if ok {
   		log.Printf("Tracker: Error occured while handling incoming connection, error: %s", err)
   	}
   }
```

In order to handle error in `defer conn.Close()` or `defer file.Close()` commands use fact that
both file and connection implement `io.Closer` interface, thus write a single function that 
handles closing Closer entities while handling an error. In this project this method is in 
util/file_util.go. 

 