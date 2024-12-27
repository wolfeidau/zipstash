package client

type ClientCmd struct {
	Save    SaveCmd    `cmd:"" help:"save a cache entry."`
	Restore RestoreCmd `cmd:"" help:"restore a cache entry."`
}
