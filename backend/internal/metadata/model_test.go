package metadata

import "testing"

func TestEpisodeFile_DownloadURL_PrefersMagnet(t *testing.T) {
	f := EpisodeFile{
		MagnetURI:  "magnet:?xt=urn:btih:abc",
		TorrentURL: "https://example.com/file.torrent",
		URL:        "https://example.com/page",
	}
	if got := f.DownloadURL(); got != f.MagnetURI {
		t.Errorf("DownloadURL() = %q, want magnet URI %q", got, f.MagnetURI)
	}
}

func TestEpisodeFile_DownloadURL_FallsBackToTorrentURL(t *testing.T) {
	f := EpisodeFile{
		TorrentURL: "https://example.com/file.torrent",
		URL:        "https://example.com/page",
	}
	if got := f.DownloadURL(); got != f.TorrentURL {
		t.Errorf("DownloadURL() = %q, want torrent URL %q", got, f.TorrentURL)
	}
}

func TestEpisodeFile_DownloadURL_FallsBackToURL(t *testing.T) {
	f := EpisodeFile{URL: "https://example.com/page"}
	if got := f.DownloadURL(); got != f.URL {
		t.Errorf("DownloadURL() = %q, want %q", got, f.URL)
	}
}

func TestEpisodeFile_DownloadURL_Empty(t *testing.T) {
	f := EpisodeFile{}
	if got := f.DownloadURL(); got != "" {
		t.Errorf("DownloadURL() = %q, want empty string", got)
	}
}
