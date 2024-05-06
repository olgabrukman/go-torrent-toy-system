## Toy project in GoLang. 

The system is a simplified bit-torrent system. There are three main components:
1. Client 
2. Tracker
3. Seeder

### Tracker
The tracker holds a list of active seeders and torrents those seeders seed.
* Receive a list of torrents from a seeder and update the internal database.
* Return a list of seeders seeding a torrent
* Has a Web API for querying and receiving a list of active seeders and how many torrents each seeder holds

### Client
A client sends a torrent request to a tracker.
* Query tracker for a torrent and download the torrent from a list of seeders.
* Support timeout in download using `context.Context`.
* Verify the downloaded file matches the `sha` signature in the torrent.

### Seeder 
A seeder sends its identifiers (host, port) and list of torrents it hosts to the tracker 
on its start-up. 

In this system, a seeder seeds file "data/aow.txt", with size 343691 bytes and
sha256 hash code 98c4a4a4710a40ae79de349b582950f9eed2196a7ff05f1e4c5aca3b51d1f588.
The client asks to get ```config/config.properties/client.fileName``` file of size 
```config/config.properties/client.fileSize``` and 
```config/config.properties/client.fileHash``` hash code. The result will be stored 
in the file ```config/config.properties/client.outputFileName```.

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

To handle errors in `defer conn.Close()` or `defer file.Close()` commands use that
both file and connection implement the `io.Closer` interface, thus you can write a single function that 
handles closing Closer entities while handling an error. In this project this method is in 
util/file_util.go. 

 
