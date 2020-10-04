package message

import (
	"fmt"
)

// Type is the message type
type Type byte

// Message in the system
type Message interface {
	Type() Type
}

// Possible message types
const (
	ChunkRequestType Type = 1 + iota
	ChunkResponseType
	TorrentType
	TorrentListType
	SeederListType
	InvalidType
)

/* iota for bitmask
const (
	Read = 1 << iota
	Write
	Execute
	ReadWrite = Read | Write
)
*/

func (t Type) String() string {
	switch t {
	case ChunkRequestType:
		return "ChunkRequest"
	case ChunkResponseType:
		return "ChunkResponse"
	case TorrentType:
		return "Torrent"
	case TorrentListType:
		return "TorrentList"
	case SeederListType:
		return "SeederList"
	}

	return fmt.Sprintf("unknown message type - %d", t)
}

// ChunkRequest to get a chunk
type ChunkRequest struct {
	FileName string
	Offset   int64
	Size     int64
}

// Type returns the message type
func (c *ChunkRequest) Type() Type {
	return ChunkRequestType
}

// ChunkResponse to get a chunk
type ChunkResponse struct {
	ChunkRequest
	Data  []byte
	Error string
}

// Type returns the message type
func (c *ChunkResponse) Type() Type {
	return ChunkResponseType
}

// Torrent message
type Torrent struct {
	Name string
	Size int64
	Hash string
}

// Type returns message type
func (t *Torrent) Type() Type {
	return TorrentType
}

// TorrentList list of torrents in host
type TorrentList struct {
	Seeder   string // host:port
	Torrents []Torrent
}

// Type returns message type
func (fl *TorrentList) Type() Type {
	return TorrentListType
}

// SeederList is list of seeders holding torrent
type SeederList struct {
	Torrent Torrent
	Seeders []string // host:port
}

// Type returns the message type
func (sl *SeederList) Type() Type {
	return SeederListType
}
