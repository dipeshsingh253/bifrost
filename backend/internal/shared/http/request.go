package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	nethttp "net/http"
	"net/netip"
	"strings"

	"github.com/gin-gonic/gin"
)

type RequestInspector struct {
	trustedCIDRs []netip.Prefix
	trustedRaw   []string
}

func NewRequestInspector(trustedProxies []string) (*RequestInspector, error) {
	inspector := &RequestInspector{
		trustedRaw: append([]string(nil), trustedProxies...),
	}

	for _, entry := range trustedProxies {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		if strings.Contains(entry, "/") {
			prefix, err := netip.ParsePrefix(entry)
			if err != nil {
				return nil, fmt.Errorf("parse trusted proxy %q: %w", entry, err)
			}
			inspector.trustedCIDRs = append(inspector.trustedCIDRs, prefix)
			continue
		}

		addr, err := netip.ParseAddr(entry)
		if err != nil {
			return nil, fmt.Errorf("parse trusted proxy %q: %w", entry, err)
		}
		bits := 32
		if addr.Is6() {
			bits = 128
		}
		inspector.trustedCIDRs = append(inspector.trustedCIDRs, netip.PrefixFrom(addr, bits))
	}

	return inspector, nil
}

func (i *RequestInspector) TrustedProxyStrings() []string {
	if i == nil {
		return nil
	}
	return append([]string(nil), i.trustedRaw...)
}

func (i *RequestInspector) IsTrustedProxy(request *nethttp.Request) bool {
	if i == nil || request == nil || len(i.trustedCIDRs) == 0 {
		return false
	}

	host, _, err := net.SplitHostPort(strings.TrimSpace(request.RemoteAddr))
	if err != nil {
		host = strings.TrimSpace(request.RemoteAddr)
	}
	addr, err := netip.ParseAddr(host)
	if err != nil {
		return false
	}

	for _, prefix := range i.trustedCIDRs {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}

func (i *RequestInspector) Scheme(request *nethttp.Request) string {
	if request == nil {
		return "http"
	}

	scheme := "http"
	if request.TLS != nil {
		scheme = "https"
	}
	if i != nil && i.IsTrustedProxy(request) {
		if forwarded := strings.TrimSpace(request.Header.Get("X-Forwarded-Proto")); forwarded != "" {
			if comma := strings.Index(forwarded, ","); comma >= 0 {
				forwarded = strings.TrimSpace(forwarded[:comma])
			}
			if forwarded == "http" || forwarded == "https" {
				scheme = forwarded
			}
		}
	}
	return scheme
}

func (i *RequestInspector) IsSecure(request *nethttp.Request) bool {
	return i.Scheme(request) == "https"
}

func (i *RequestInspector) DefaultBackendURL(request *nethttp.Request) string {
	if request == nil || strings.TrimSpace(request.Host) == "" {
		return ""
	}
	return i.Scheme(request) + "://" + request.Host
}

func DecodeJSON(c *gin.Context, dst any) bool {
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		writeDecodeError(c, err)
		return false
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		c.JSON(nethttp.StatusBadRequest, Error("request body must contain a single JSON object", "INVALID_REQUEST", nil))
		return false
	}

	return true
}

func writeDecodeError(c *gin.Context, err error) {
	var maxBytesErr *nethttp.MaxBytesError
	switch {
	case errors.As(err, &maxBytesErr):
		c.JSON(nethttp.StatusRequestEntityTooLarge, Error("request body exceeds the allowed size", "REQUEST_TOO_LARGE", gin.H{
			"limit_bytes": maxBytesErr.Limit,
		}))
	case errors.Is(err, io.EOF):
		c.JSON(nethttp.StatusBadRequest, Error("request body is required", "INVALID_REQUEST", nil))
	default:
		c.JSON(nethttp.StatusBadRequest, Error("invalid request body", "INVALID_REQUEST", gin.H{"reason": err.Error()}))
	}
}
