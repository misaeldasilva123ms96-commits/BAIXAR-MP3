package core

import (
	"errors"
	"net/netip"
	"net/url"
	"path/filepath"
	"strings"
)

func (settings Settings) Validate() error {
	if _, ok := QualityArguments()[settings.DefaultQuality]; !ok {
		return errors.New("qualidade padrão inválida")
	}
	if settings.DownloadDirectory == "" || !filepath.IsAbs(settings.DownloadDirectory) || strings.ContainsRune(settings.DownloadDirectory, '\x00') {
		return errors.New("pasta de downloads deve ser um caminho absoluto")
	}
	return nil
}

var allowedMediaHosts = map[string]bool{
	"youtube.com": true, "www.youtube.com": true, "m.youtube.com": true,
	"music.youtube.com": true, "youtube-nocookie.com": true,
	"www.youtube-nocookie.com": true, "youtu.be": true,
}

var blockedMediaPaths = []string{"/redirect", "/attribution_link", "/oembed"}

func ValidateMediaURL(raw string) (*url.URL, error) {
	if len(raw) == 0 || len(raw) > 2048 {
		return nil, errors.New("URL inválida")
	}
	u, err := url.ParseRequestURI(strings.TrimSpace(raw))
	if err != nil || u.Hostname() == "" {
		return nil, errors.New("URL inválida")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, errors.New("protocolo não permitido")
	}
	if u.User != nil {
		return nil, errors.New("credenciais na URL não são permitidas")
	}
	host := strings.TrimSuffix(strings.ToLower(u.Hostname()), ".")
	if !allowedMediaHosts[host] {
		return nil, errors.New("somente URLs oficiais do YouTube são aceitas")
	}
	path := strings.ToLower(strings.TrimSuffix(u.EscapedPath(), "/"))
	for _, blocked := range blockedMediaPaths {
		if path == blocked || strings.HasPrefix(path, blocked+"/") {
			return nil, errors.New("endpoint de redirecionamento não permitido")
		}
	}
	if ip, err := netip.ParseAddr(host); err == nil && !AddressIsPublic(ip) {
		return nil, errors.New("endereço privado não permitido")
	}
	return u, nil
}

func AddressIsPublic(ip netip.Addr) bool {
	return ip.IsValid() && !ip.IsPrivate() && !ip.IsLoopback() && !ip.IsLinkLocalUnicast() &&
		!ip.IsLinkLocalMulticast() && !ip.IsMulticast() && !ip.IsUnspecified()
}

func validateRequest(request DownloadRequest) error {
	if _, err := ValidateMediaURL(request.URL); err != nil {
		return err
	}
	if _, ok := QualityArguments()[request.Quality]; !ok {
		return errors.New("qualidade inválida")
	}
	if request.PlaylistStart < 0 || request.PlaylistEnd < 0 {
		return errors.New("intervalo inválido")
	}
	if request.PlaylistStart > 0 && request.PlaylistEnd > 0 && request.PlaylistStart > request.PlaylistEnd {
		return errors.New("início maior que o fim")
	}
	if request.PlaylistStart > 500 || request.PlaylistEnd > 500 {
		return errors.New("limite máximo de playlist excedido")
	}
	return nil
}
