package eos

import "time"

type Config struct {
	SSHTarget         string
	Timeout           time.Duration
	AcceptNewHostKeys bool
}

type Client struct {
	// sshTarget is the gateway/initial SSH host supplied by the user (e.g. "eospilot").
	sshTarget string
	// resolvedSSHTarget is set after MGM master discovery and becomes the
	// effective destination for all subsequent commands.  When empty,
	// sshTarget is used as-is.
	resolvedSSHTarget string
	timeout           time.Duration
	acceptNewHostKeys bool
	// sessionLogPath is the log file for this specific session, set once at
	// construction time.  Empty means logging is disabled (e.g. home dir error).
	sessionLogPath string
}

type EntryKind string

const (
	EntryKindFile      EntryKind = "file"
	EntryKindContainer EntryKind = "dir"
)

type Entry struct {
	Kind           EntryKind
	Name           string
	Path           string
	ID             uint64
	ParentID       uint64
	Inode          uint64
	UID            uint32
	GID            uint32
	Size           uint64
	TreeSize       int64
	Files          uint64
	Containers     uint64
	LayoutID       uint32
	Flags          uint32
	Mode           uint32
	Locations      int
	LinkName       string
	ETag           string
	ModifiedAt     time.Time
	ChangedAt      time.Time
	SynchronizedAt time.Time
}

type Directory struct {
	Path    string
	Self    Entry
	Entries []Entry
}

type NamespaceAttr struct {
	Key   string
	Value string
}

type NodeStats struct {
	State       string
	FileCount   uint64
	DirCount    uint64
	BootTime    time.Time
	Uptime      time.Duration
	CurrentFID  uint64
	CurrentCID  uint64
	MemVirtual  uint64
	MemResident uint64
	MemShared   uint64
	MemGrowth   uint64
	ThreadCount uint64
	FileDescs   uint64
}

type FstRecord struct {
	Type            string
	Host            string
	Port            int
	Geotag          string
	Status          string
	Activated       string
	HeartbeatDelta  int64
	FileSystemCount int
	EOSVersion      string
	Kernel          string
	Uptime          string
	ThreadCount     uint64
	RSSBytes        uint64
	VSizeBytes      uint64
	CapacityBytes   uint64
	UsedBytes       uint64
	FreeBytes       uint64
	UsedFiles       uint64
	DiskLoad        float64
	ReadRateMB      float64
	WriteRateMB     float64
}

type MgmRecord struct {
	Host       string
	Port       int
	QDBHost    string
	QDBPort    int
	Role       string
	Geotag     string
	Status     string
	Heartbeat  string
	EOSVersion string
}

type FileSystemRecord struct {
	Host          string
	Port          uint64
	ID            uint64
	Path          string
	SchedGroup    string
	Geotag        string
	Boot          string
	ConfigStatus  string
	DrainStatus   string
	Active        string
	Health        string
	CapacityBytes uint64
	UsedBytes     uint64
	FreeBytes     uint64
	UsedFiles     uint64
	DiskBWMB      float64
	DiskIOPS      float64
	ReadRateMB    float64
	WriteRateMB   float64
}

type GroupRecord struct {
	Name          string
	Status        string
	NoFS          int
	CapacityBytes uint64
	UsedBytes     uint64
	FreeBytes     uint64
	NumFiles      uint64
}

type SpaceRecord struct {
	Name          string
	Type          string
	Status        string
	Groups        uint64
	NumFiles      uint64
	NumContainers uint64
	CapacityBytes uint64
	UsedBytes     uint64
	FreeBytes     uint64
}

type SpaceStatusRecord struct {
	Key   string
	Value string
}

type NamespaceStats struct {
	MasterHost              string
	TotalFiles              uint64
	TotalDirectories        uint64
	CurrentFID              uint64
	CurrentCID              uint64
	GeneratedFID            uint64
	GeneratedCID            uint64
	ContentionRead          float64
	ContentionWrite         float64
	CacheFilesMax           uint64
	CacheFilesOccup         uint64
	CacheFilesRequests      uint64
	CacheFilesHits          uint64
	CacheContainersMax      uint64
	CacheContainersOccup    uint64
	CacheContainersRequests uint64
	CacheContainersHits     uint64
}

type IOShapingMode int

const (
	IOShapingApps IOShapingMode = iota
	IOShapingUsers
	IOShapingGroups
)

type IOShapingRecord struct {
	ID        string
	Type      string
	WindowSec int
	ReadBps   float64
	WriteBps  float64
	ReadIOPS  float64
	WriteIOPS float64
}

type IOShapingPolicyRecord struct {
	ID                          string
	Type                        string
	Enabled                     bool
	LimitReadBytesPerSec        float64
	LimitWriteBytesPerSec       float64
	ReservationReadBytesPerSec  float64
	ReservationWriteBytesPerSec float64
}

// raftReplica holds the parsed status of a single replica node from raft-info.
type raftReplica struct {
	Host    string
	Status  string // ONLINE or OFFLINE
	Sync    string // UP-TO-DATE or LAGGING
	Version string
}

// raftInfo holds parsed output of `redis-cli -p 7777 raft-info`.
type raftInfo struct {
	Leader    string
	Myself    string
	MyRole    string
	MyVersion string
	MyHealth  string
	Nodes     []string
	Replicas  []raftReplica
}

// cliFileInfo is the JSON structure returned by `eos -j fileinfo`.
type cliFileInfo struct {
	Children       []cliFileInfo `json:"children"`
	CTime          int64         `json:"ctime"`
	CTimeNS        int64         `json:"ctime_ns"`
	ETag           string        `json:"etag"`
	Flags          uint32        `json:"flags"`
	GID            uint32        `json:"gid"`
	ID             uint64        `json:"id"`
	Inode          uint64        `json:"inode"`
	Locations      []cliLocation `json:"locations"`
	Mode           uint32        `json:"mode"`
	MTime          int64         `json:"mtime"`
	MTimeNS        int64         `json:"mtime_ns"`
	Name           string        `json:"name"`
	NFiles         uint64        `json:"nfiles"`
	NNDirectories  uint64        `json:"nndirectories"`
	Path           string        `json:"path"`
	PID            uint64        `json:"pid"`
	Size           uint64        `json:"size"`
	TreeContainers uint64        `json:"treecontainers"`
	TreeFiles      uint64        `json:"treefiles"`
	TreeSize       int64         `json:"treesize"`
	UID            uint32        `json:"uid"`
	LinkTarget     string        `json:"link"`
}

type cliLocation struct {
	FSID uint64 `json:"fsid"`
}
