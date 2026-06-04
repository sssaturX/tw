package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/adeithe/go-twitch/irc"
)

type dialContextFunc func(context.Context, string, string) (net.Conn, error)

func accountHTTPClient(acc Account) (*http.Client, error) {
	dialContext, err := accountDialContext(acc)
	if err != nil {
		return nil, err
	}
	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			DialContext:           dialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          20,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: time.Second,
		},
	}, nil
}

func accountDialContext(acc Account) (dialContextFunc, error) {
	if strings.TrimSpace(acc.SOCKS5Proxy) == "" {
		var dialer net.Dialer
		return dialer.DialContext, nil
	}
	proxyAddr, username, password, err := parseSOCKS5Proxy(acc.SOCKS5Proxy)
	if err != nil {
		return nil, err
	}

	return func(ctx context.Context, network, address string) (net.Conn, error) {
		if network != "tcp" && network != "tcp4" && network != "tcp6" {
			return nil, fmt.Errorf("socks5 proxy supports tcp only, got %s", network)
		}
		var dialer net.Dialer
		conn, err := dialer.DialContext(ctx, "tcp", proxyAddr)
		if err != nil {
			return nil, err
		}
		if deadline, ok := ctx.Deadline(); ok {
			_ = conn.SetDeadline(deadline)
			defer func() { _ = conn.SetDeadline(time.Time{}) }()
		}
		if err := socks5Connect(conn, address, username, password); err != nil {
			_ = conn.Close()
			return nil, err
		}
		return conn, nil
	}, nil
}

func parseSOCKS5Proxy(raw string) (addr, username, password string, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", "", errors.New("empty socks5 proxy")
	}
	if !strings.Contains(raw, "://") {
		raw = "socks5://" + raw
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return "", "", "", err
	}
	if parsed.Scheme != "socks5" && parsed.Scheme != "socks5h" {
		return "", "", "", fmt.Errorf("unsupported proxy scheme %q", parsed.Scheme)
	}
	if parsed.Host == "" {
		return "", "", "", errors.New("socks5 proxy host is required")
	}
	if _, _, err := net.SplitHostPort(parsed.Host); err != nil {
		return "", "", "", fmt.Errorf("socks5 proxy must include host:port: %w", err)
	}
	if parsed.User != nil {
		username = parsed.User.Username()
		password, _ = parsed.User.Password()
	}
	return parsed.Host, username, password, nil
}

func socks5Connect(conn net.Conn, target, username, password string) error {
	methods := []byte{0x00}
	if username != "" || password != "" {
		methods = append(methods, 0x02)
	}
	if _, err := conn.Write([]byte{0x05, byte(len(methods))}); err != nil {
		return err
	}
	if _, err := conn.Write(methods); err != nil {
		return err
	}

	choice := []byte{0, 0}
	if _, err := io.ReadFull(conn, choice); err != nil {
		return err
	}
	if choice[0] != 0x05 {
		return errors.New("invalid socks5 version in method response")
	}
	switch choice[1] {
	case 0x00:
	case 0x02:
		if err := socks5UsernamePasswordAuth(conn, username, password); err != nil {
			return err
		}
	case 0xff:
		return errors.New("socks5 proxy rejected all auth methods")
	default:
		return fmt.Errorf("unsupported socks5 auth method 0x%02x", choice[1])
	}

	host, portText, err := net.SplitHostPort(target)
	if err != nil {
		return err
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		return err
	}
	if port < 1 || port > 65535 {
		return fmt.Errorf("invalid target port %d", port)
	}

	request := []byte{0x05, 0x01, 0x00}
	if ip := net.ParseIP(host); ip != nil {
		if v4 := ip.To4(); v4 != nil {
			request = append(request, 0x01)
			request = append(request, v4...)
		} else {
			request = append(request, 0x04)
			request = append(request, ip.To16()...)
		}
	} else {
		if len(host) > 255 {
			return errors.New("target hostname is too long for socks5")
		}
		request = append(request, 0x03, byte(len(host)))
		request = append(request, []byte(host)...)
	}
	request = binary.BigEndian.AppendUint16(request, uint16(port))
	if _, err := conn.Write(request); err != nil {
		return err
	}

	header := []byte{0, 0, 0, 0}
	if _, err := io.ReadFull(conn, header); err != nil {
		return err
	}
	if header[0] != 0x05 {
		return errors.New("invalid socks5 version in connect response")
	}
	if header[1] != 0x00 {
		return fmt.Errorf("socks5 connect failed: 0x%02x", header[1])
	}

	var skip int
	switch header[3] {
	case 0x01:
		skip = 4
	case 0x03:
		length := []byte{0}
		if _, err := io.ReadFull(conn, length); err != nil {
			return err
		}
		skip = int(length[0])
	case 0x04:
		skip = 16
	default:
		return fmt.Errorf("invalid socks5 address type 0x%02x", header[3])
	}
	if skip > 0 {
		if _, err := io.CopyN(io.Discard, conn, int64(skip)); err != nil {
			return err
		}
	}
	if _, err := io.CopyN(io.Discard, conn, 2); err != nil {
		return err
	}
	return nil
}

func socks5UsernamePasswordAuth(conn net.Conn, username, password string) error {
	if len(username) > 255 || len(password) > 255 {
		return errors.New("socks5 username/password must be <= 255 bytes")
	}
	req := []byte{0x01, byte(len(username))}
	req = append(req, []byte(username)...)
	req = append(req, byte(len(password)))
	req = append(req, []byte(password)...)
	if _, err := conn.Write(req); err != nil {
		return err
	}
	res := []byte{0, 0}
	if _, err := io.ReadFull(conn, res); err != nil {
		return err
	}
	if res[1] != 0x00 {
		return errors.New("socks5 username/password auth failed")
	}
	return nil
}

type twitchIRCConn struct {
	username string
	token    string
	dial     dialContextFunc
	conn     net.Conn
	writerMu sync.Mutex

	onMessage      []func(irc.ChatMessage)
	onServerNotice []func(irc.ServerNotice)
}

func newTwitchIRCConn(acc Account) (*twitchIRCConn, error) {
	dialContext, err := accountDialContext(acc)
	if err != nil {
		return nil, err
	}
	return &twitchIRCConn{
		username: normalizeLogin(acc.Login),
		token:    sanitizeToken(acc.Token),
		dial:     dialContext,
	}, nil
}

func (c *twitchIRCConn) Connect(ctx context.Context) error {
	if c.conn != nil {
		return nil
	}
	rawConn, err := c.dial(ctx, "tcp", irc.IP+":6697")
	if err != nil {
		return err
	}
	tlsConn := tls.Client(rawConn, &tls.Config{ServerName: irc.IP, MinVersion: tls.VersionTLS12})
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		_ = rawConn.Close()
		return err
	}
	c.conn = tlsConn
	go c.reader()

	username := c.username
	token := c.token
	if username == "" || token == "" {
		username = "justinfan12345"
		token = "Kappa123"
	}
	return c.SendRaw(
		"CAP REQ :twitch.tv/membership twitch.tv/tags twitch.tv/commands",
		"PASS oauth:"+token,
		"NICK "+username,
	)
}

func (c *twitchIRCConn) Join(channel string) error {
	return c.SendRaw("JOIN #" + normalizeLogin(channel))
}

func (c *twitchIRCConn) Say(channel string, message string) error {
	if c.username == "" || c.token == "" {
		return errors.New("irc connection is not authenticated")
	}
	return c.SendRaw(fmt.Sprintf("PRIVMSG #%s :%s", normalizeLogin(channel), message))
}

func (c *twitchIRCConn) SayReply(channel string, message string, replyToID string) error {
	if c.username == "" || c.token == "" {
		return errors.New("irc connection is not authenticated")
	}
	replyToID = sanitizeIRCTagValue(replyToID)
	if replyToID == "" {
		return c.Say(channel, message)
	}
	return c.SendRaw(fmt.Sprintf("@reply-parent-msg-id=%s PRIVMSG #%s :%s", replyToID, normalizeLogin(channel), message))
}

func sanitizeIRCTagValue(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, " ", "")
	value = strings.ReplaceAll(value, "\r", "")
	value = strings.ReplaceAll(value, "\n", "")
	value = strings.ReplaceAll(value, ";", "")
	return value
}

func (c *twitchIRCConn) SendRaw(raw ...string) error {
	if c.conn == nil {
		return errors.New("irc connection is not connected")
	}
	c.writerMu.Lock()
	defer c.writerMu.Unlock()
	for _, msg := range raw {
		if _, err := c.conn.Write([]byte(msg + "\r\n")); err != nil {
			return err
		}
	}
	return nil
}

func (c *twitchIRCConn) OnMessage(f func(irc.ChatMessage)) {
	c.onMessage = append(c.onMessage, f)
}

func (c *twitchIRCConn) OnServerNotice(f func(irc.ServerNotice)) {
	c.onServerNotice = append(c.onServerNotice, f)
}

func (c *twitchIRCConn) Close() {
	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}
}

func (c *twitchIRCConn) reader() {
	scanner := bufio.NewScanner(c.conn)
	for scanner.Scan() {
		line := scanner.Text()
		msg, err := irc.NewParsedMessage(line)
		if err != nil {
			continue
		}
		switch msg.Command {
		case irc.CMDPing:
			_ = c.SendRaw("PONG :tmi.twitch.tv")
		case irc.CMDPrivMessage:
			chatMsg := irc.NewChatMessage(msg)
			for _, f := range c.onMessage {
				go f(chatMsg)
			}
		case irc.CMDNotice:
			notice := irc.NewServerNotice(msg)
			for _, f := range c.onServerNotice {
				go f(notice)
			}
		}
	}
}
