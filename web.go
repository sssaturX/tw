package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/adeithe/go-twitch/api"
	"github.com/adeithe/go-twitch/irc"
)

type webSession struct {
	configPath string
	cfg        Config
	channel    string

	mu       sync.Mutex
	resolved map[string]resolvedAccount
	current  resolvedAccount
	writer   *twitchIRCConn
	reader   *twitchIRCConn

	clients      map[chan webEvent]bool
	clientsMu    sync.Mutex
	chatMessages []chatPayload

	autoMu          sync.Mutex
	autoID          int
	autoCancel      context.CancelFunc
	autoRunning     bool
	autoAccounts    []string
	autoMinSeconds  int
	autoMaxSeconds  int
	autoFileLines   int
	autoSent        int
	autoLastAccount string
	autoLastMessage string
	autoNextAt      time.Time
	autoError       string
}

type webEvent struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

type accountPayload struct {
	Name        string `json:"name"`
	Login       string `json:"login"`
	UserID      string `json:"user_id,omitempty"`
	HasProxy    bool   `json:"has_proxy"`
	SOCKS5Proxy string `json:"socks5_proxy,omitempty"`
	Active      bool   `json:"active"`
}

type chatPayload struct {
	ID          string `json:"id,omitempty"`
	Time        string `json:"time"`
	Channel     string `json:"channel"`
	Login       string `json:"login"`
	DisplayName string `json:"display_name"`
	Text        string `json:"text"`
	Outgoing    bool   `json:"outgoing"`
	System      bool   `json:"system"`
}

type autoSenderPayload struct {
	Running     bool     `json:"running"`
	Accounts    []string `json:"accounts"`
	MinSeconds  int      `json:"min_seconds"`
	MaxSeconds  int      `json:"max_seconds"`
	File        string   `json:"file"`
	FileLines   int      `json:"file_lines"`
	Sent        int      `json:"sent"`
	LastAccount string   `json:"last_account,omitempty"`
	LastMessage string   `json:"last_message,omitempty"`
	NextAt      string   `json:"next_at,omitempty"`
	Error       string   `json:"error,omitempty"`
}

const autoSenderFileName = "auto-sender.txt"

func webCommand(args []string) error {
	fs := flag.NewFlagSet("web", flag.ContinueOnError)
	configPath := fs.String("config", defaultConfigPath, "path to accounts config")
	accountName := fs.String("account", "", "initial account name/login")
	channel := fs.String("channel", "", "target channel login")
	listen := fs.String("listen", "127.0.0.1:8080", "listen address")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *channel == "" {
		return errors.New("--channel is required")
	}

	cfg, err := loadOptionalConfig(*configPath)
	if err != nil {
		return err
	}
	initial := *accountName
	if strings.TrimSpace(initial) == "" && len(cfg.Accounts) > 0 {
		initial = cfg.Accounts[0].Name
	}

	session := &webSession{
		configPath: *configPath,
		cfg:        cfg,
		channel:    normalizeLogin(*channel),
		resolved:   map[string]resolvedAccount{},
		clients:    map[chan webEvent]bool{},
	}
	ctx := context.Background()
	if initial != "" {
		if err := session.switchAccount(ctx, initial); err != nil {
			session.addChat(chatPayload{
				Time:   time.Now().Format("15:04:05"),
				Text:   "account connect failed: " + err.Error(),
				System: true,
			})
		}
	}
	if err := session.startReader(ctx); err != nil {
		session.addChat(chatPayload{
			Time:   time.Now().Format("15:04:05"),
			Text:   "chat reader failed: " + err.Error(),
			System: true,
		})
	}
	defer session.close()

	mux := http.NewServeMux()
	mux.HandleFunc("/", session.handleIndex)
	mux.HandleFunc("/events", session.handleEvents)
	mux.HandleFunc("/api/state", session.handleState)
	mux.HandleFunc("/api/accounts", session.handleAccounts)
	mux.HandleFunc("/api/channel", session.handleChannelSwitch)
	mux.HandleFunc("/api/account", session.handleAccountSwitch)
	mux.HandleFunc("/api/account/proxy", session.handleAccountProxy)
	mux.HandleFunc("/api/send", session.handleSend)
	mux.HandleFunc("/api/presets", session.handlePresets)
	mux.HandleFunc("/api/emotes", session.handleEmotes)
	mux.HandleFunc("/api/rewards", session.handleRewards)
	mux.HandleFunc("/api/auto-sender", session.handleAutoSender)

	fmt.Printf("web UI: http://%s\n", *listen)
	return http.ListenAndServe(*listen, mux)
}

func (s *webSession) startReader(ctx context.Context) error {
	s.mu.Lock()
	channel := s.channel
	s.mu.Unlock()

	reader, err := newTwitchIRCConn(Account{})
	if err != nil {
		return err
	}
	reader.OnMessage(func(msg irc.ChatMessage) {
		displayName := msg.Sender.DisplayName
		if displayName == "" {
			displayName = msg.Sender.Username
		}
		s.addChat(chatPayload{
			ID:          msg.ID,
			Time:        msg.CreatedAt.Format("15:04:05"),
			Channel:     msg.Channel,
			Login:       msg.Sender.Username,
			DisplayName: displayName,
			Text:        msg.Text,
		})
	})
	reader.OnServerNotice(func(notice irc.ServerNotice) {
		s.addChat(chatPayload{
			Time:   time.Now().Format("15:04:05"),
			Text:   notice.Message,
			System: true,
		})
	})
	if err := reader.Connect(ctx); err != nil {
		return err
	}
	if err := reader.Join(channel); err != nil {
		reader.Close()
		return err
	}
	s.mu.Lock()
	oldReader := s.reader
	s.reader = reader
	s.mu.Unlock()
	if oldReader != nil {
		oldReader.Close()
	}
	return nil
}

func (s *webSession) switchAccount(ctx context.Context, name string) error {
	s.mu.Lock()
	acc, index, err := s.findAccountLocked(name)
	s.mu.Unlock()
	if err != nil {
		return err
	}

	resolved, err := s.resolveAccount(ctx, acc)
	if err != nil {
		return err
	}

	s.mu.Lock()
	channel := s.channel
	s.mu.Unlock()

	nextWriter, err := newLiveWriter(ctx, resolved, channel)
	if err != nil {
		return err
	}

	s.mu.Lock()
	oldWriter := s.writer
	s.writer = nextWriter
	s.current = resolved
	s.cfg.Accounts[index].Login = resolved.Login
	s.cfg.Accounts[index].ClientID = resolved.ClientID
	s.cfg.Accounts[index].UserID = resolved.UserID
	s.mu.Unlock()

	if oldWriter != nil {
		oldWriter.Close()
	}

	s.broadcast(webEvent{Type: "state", Data: s.statePayload()})
	s.addChat(chatPayload{
		Time:   time.Now().Format("15:04:05"),
		Text:   fmt.Sprintf("active account: %s (%s)", resolved.Name, resolved.Login),
		System: true,
	})
	return nil
}

func (s *webSession) switchChannel(ctx context.Context, channel string) error {
	channel = normalizeLogin(channel)
	if channel == "" {
		return errors.New("channel is required")
	}

	s.stopAutoSender()

	s.mu.Lock()
	if channel == s.channel {
		s.mu.Unlock()
		return nil
	}
	current := s.current
	oldWriter := s.writer
	oldReader := s.reader
	s.channel = channel
	s.writer = nil
	s.reader = nil
	s.chatMessages = nil
	s.mu.Unlock()

	if oldWriter != nil {
		oldWriter.Close()
	}
	if oldReader != nil {
		oldReader.Close()
	}

	if err := s.startReader(ctx); err != nil {
		s.addChat(chatPayload{
			Time:   time.Now().Format("15:04:05"),
			Text:   "chat reader failed: " + err.Error(),
			System: true,
		})
	}

	if current.Name != "" {
		writer, err := newLiveWriter(ctx, current, channel)
		if err != nil {
			s.addChat(chatPayload{
				Time:   time.Now().Format("15:04:05"),
				Text:   "writer reconnect failed: " + err.Error(),
				System: true,
			})
		} else {
			s.mu.Lock()
			s.writer = writer
			s.mu.Unlock()
		}
	}

	s.broadcast(webEvent{Type: "state", Data: s.statePayload()})
	s.addChat(chatPayload{
		Time:   time.Now().Format("15:04:05"),
		Text:   "connected to #" + channel,
		System: true,
	})
	return nil
}

func (s *webSession) resolveAccount(ctx context.Context, acc Account) (resolvedAccount, error) {
	cacheKey := strings.ToLower(acc.Name)
	s.mu.Lock()
	resolved, ok := s.resolved[cacheKey]
	s.mu.Unlock()
	if ok {
		return resolved, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	resolved, err := validateAccount(ctx, acc)
	if err != nil {
		return resolvedAccount{}, err
	}
	s.mu.Lock()
	s.resolved[cacheKey] = resolved
	if resolved.Login != "" {
		s.resolved[strings.ToLower(resolved.Login)] = resolved
	}
	s.mu.Unlock()
	return resolved, nil
}

func (s *webSession) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(webHTML))
}

func (s *webSession) handleEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := make(chan webEvent, 32)
	s.clientsMu.Lock()
	s.clients[ch] = true
	s.clientsMu.Unlock()
	defer func() {
		s.clientsMu.Lock()
		delete(s.clients, ch)
		s.clientsMu.Unlock()
		close(ch)
	}()

	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	s.sendSSE(w, webEvent{Type: "state", Data: s.statePayload()})
	for _, msg := range s.history() {
		s.sendSSE(w, webEvent{Type: "chat", Data: msg})
	}

	for {
		select {
		case <-r.Context().Done():
			return
		case event := <-ch:
			s.sendSSE(w, event)
		}
	}
}

func (s *webSession) sendSSE(w http.ResponseWriter, event webEvent) {
	bs, err := json.Marshal(event)
	if err != nil {
		return
	}
	_, _ = fmt.Fprintf(w, "data: %s\n\n", bs)
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (s *webSession) handleState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, s.statePayload())
}

func (s *webSession) handleAccountSwitch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.switchAccount(r.Context(), req.Name); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, s.statePayload())
}

func (s *webSession) handleAccounts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req Account
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Login = normalizeLogin(req.Login)
	req.Token = sanitizeToken(req.Token)
	req.SOCKS5Proxy = strings.TrimSpace(req.SOCKS5Proxy)
	if req.Name == "" {
		if req.Login != "" {
			req.Name = req.Login
		} else {
			http.Error(w, "name or login is required", http.StatusBadRequest)
			return
		}
	}
	if req.Token == "" {
		http.Error(w, "token is required", http.StatusBadRequest)
		return
	}
	if req.SOCKS5Proxy != "" {
		if _, _, _, err := parseSOCKS5Proxy(req.SOCKS5Proxy); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	s.mu.Lock()
	for _, acc := range s.cfg.Accounts {
		if strings.EqualFold(acc.Name, req.Name) || (req.Login != "" && strings.EqualFold(acc.Login, req.Login)) {
			s.mu.Unlock()
			http.Error(w, "account already exists", http.StatusBadRequest)
			return
		}
	}
	s.cfg.Accounts = append(s.cfg.Accounts, req)
	shouldSwitch := s.current.Name == ""
	s.mu.Unlock()

	if err := saveConfig(s.configPath, s.cfg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if shouldSwitch {
		if err := s.switchAccount(r.Context(), req.Name); err != nil {
			s.addChat(chatPayload{
				Time:   time.Now().Format("15:04:05"),
				Text:   "account saved, connect failed: " + err.Error(),
				System: true,
			})
		}
	}

	s.broadcast(webEvent{Type: "state", Data: s.statePayload()})
	writeJSON(w, s.statePayload())
}

func (s *webSession) handleChannelSwitch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Channel string `json:"channel"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.switchChannel(r.Context(), req.Channel); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, s.statePayload())
}

func (s *webSession) handleAccountProxy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Name  string `json:"name"`
		Proxy string `json:"proxy"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	req.Proxy = strings.TrimSpace(req.Proxy)
	if req.Proxy != "" {
		if _, _, _, err := parseSOCKS5Proxy(req.Proxy); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	s.mu.Lock()
	acc, index, err := s.findAccountLocked(req.Name)
	if err != nil {
		s.mu.Unlock()
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.cfg.Accounts[index].SOCKS5Proxy = req.Proxy
	delete(s.resolved, strings.ToLower(acc.Name))
	delete(s.resolved, strings.ToLower(acc.Login))
	currentName := s.current.Name
	s.mu.Unlock()

	if err := saveConfig(s.configPath, s.cfg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if strings.EqualFold(currentName, req.Name) || strings.EqualFold(acc.Login, req.Name) {
		if err := s.switchAccount(r.Context(), req.Name); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	s.broadcast(webEvent{Type: "state", Data: s.statePayload()})
	writeJSON(w, s.statePayload())
}

func (s *webSession) handleSend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Message string `json:"message"`
		ReplyTo string `json:"reply_to"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	req.Message = strings.TrimSpace(req.Message)
	if req.Message == "" {
		http.Error(w, "message is required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	writer := s.writer
	channel := s.channel
	s.mu.Unlock()
	if writer == nil {
		http.Error(w, "no active writer", http.StatusBadRequest)
		return
	}
	if err := writer.SayReply(channel, req.Message, req.ReplyTo); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

func (s *webSession) handlePresets(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var req Preset
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		req.ID = strings.TrimSpace(req.ID)
		req.Title = strings.TrimSpace(req.Title)
		req.Message = strings.TrimSpace(req.Message)
		if req.Message == "" {
			http.Error(w, "message is required", http.StatusBadRequest)
			return
		}
		if req.ID == "" {
			req.ID = fmt.Sprintf("preset_%d", time.Now().UnixNano())
		}
		if req.Title == "" {
			req.Title = req.Message
		}

		s.mu.Lock()
		for _, preset := range s.cfg.Presets {
			if strings.EqualFold(preset.ID, req.ID) {
				s.mu.Unlock()
				http.Error(w, "preset already exists", http.StatusBadRequest)
				return
			}
		}
		s.cfg.Presets = append(s.cfg.Presets, req)
		cfg := s.cfg
		s.mu.Unlock()

		if err := saveConfig(s.configPath, cfg); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.broadcast(webEvent{Type: "state", Data: s.statePayload()})
		writeJSON(w, s.statePayload())
	case http.MethodDelete:
		id := strings.TrimSpace(r.URL.Query().Get("id"))
		if id == "" {
			http.Error(w, "id is required", http.StatusBadRequest)
			return
		}

		s.mu.Lock()
		next := s.cfg.Presets[:0]
		found := false
		for _, preset := range s.cfg.Presets {
			if strings.EqualFold(preset.ID, id) {
				found = true
				continue
			}
			next = append(next, preset)
		}
		s.cfg.Presets = next
		cfg := s.cfg
		s.mu.Unlock()
		if !found {
			http.Error(w, "preset not found", http.StatusBadRequest)
			return
		}

		if err := saveConfig(s.configPath, cfg); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.broadcast(webEvent{Type: "state", Data: s.statePayload()})
		writeJSON(w, s.statePayload())
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *webSession) handleEmotes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	kind := r.URL.Query().Get("kind")
	if kind == "" {
		kind = "channel"
	}
	s.mu.Lock()
	current := s.current
	channel := s.channel
	s.mu.Unlock()

	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	emotes, err := listEmotes(ctx, current, kind, channel)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if len(emotes) > 80 {
		emotes = emotes[:80]
	}
	writeJSON(w, emotes)
}

func (s *webSession) handleRewards(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.mu.Lock()
	current := s.current
	channel := s.channel
	s.mu.Unlock()

	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	rewards, err := listCustomRewards(ctx, current, channel, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, rewards)
}

func (s *webSession) handleAutoSender(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, s.autoSenderPayloadWithFile())
	case http.MethodPost:
		var req struct {
			Accounts   []string `json:"accounts"`
			MinSeconds int      `json:"min_seconds"`
			MaxSeconds int      `json:"max_seconds"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		accountNames, err := s.normalizeAutoAccounts(req.Accounts)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		lines, err := readAutoSenderLines(autoSenderFileName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if len(lines) == 0 {
			http.Error(w, autoSenderFileName+" has no messages", http.StatusBadRequest)
			return
		}

		minSeconds := req.MinSeconds
		maxSeconds := req.MaxSeconds
		if minSeconds < 1 {
			minSeconds = 1
		}
		if maxSeconds < minSeconds {
			maxSeconds = minSeconds
		}

		s.stopAutoSender()
		ctx, cancel := context.WithCancel(context.Background())

		s.autoMu.Lock()
		s.autoID++
		autoID := s.autoID
		s.autoCancel = cancel
		s.autoRunning = true
		s.autoAccounts = append([]string(nil), accountNames...)
		s.autoMinSeconds = minSeconds
		s.autoMaxSeconds = maxSeconds
		s.autoFileLines = len(lines)
		s.autoSent = 0
		s.autoLastAccount = ""
		s.autoLastMessage = ""
		s.autoNextAt = time.Time{}
		s.autoError = ""
		payload := s.autoSenderPayloadLocked()
		s.autoMu.Unlock()

		s.broadcast(webEvent{Type: "auto", Data: payload})
		s.addChat(chatPayload{
			Time:   time.Now().Format("15:04:05"),
			Text:   fmt.Sprintf("auto-sender started: %d messages, %d account(s), %d-%d sec", len(lines), len(accountNames), minSeconds, maxSeconds),
			System: true,
		})
		go s.runAutoSender(ctx, autoID, accountNames, lines, minSeconds, maxSeconds)
		writeJSON(w, s.autoSenderPayload())
	case http.MethodDelete:
		s.stopAutoSender()
		writeJSON(w, s.autoSenderPayload())
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *webSession) normalizeAutoAccounts(names []string) ([]string, error) {
	seen := map[string]bool{}
	out := make([]string, 0, len(names))
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, name := range names {
		acc, _, err := s.findAccountLocked(name)
		if err != nil {
			return nil, err
		}
		key := strings.ToLower(acc.Name)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, acc.Name)
	}
	if len(out) == 0 {
		return nil, errors.New("choose at least one account")
	}
	return out, nil
}

func readAutoSenderLines(path string) ([]string, error) {
	bs, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%s not found", path)
		}
		return nil, err
	}
	rawLines := strings.Split(string(bs), "\n")
	lines := make([]string, 0, len(rawLines))
	for _, line := range rawLines {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		lines = append(lines, line)
	}
	return lines, nil
}

func (s *webSession) runAutoSender(ctx context.Context, autoID int, accounts, messages []string, minSeconds, maxSeconds int) {
	writers := map[string]*twitchIRCConn{}
	defer func() {
		for _, writer := range writers {
			writer.Close()
		}
		s.autoMu.Lock()
		if s.autoID == autoID {
			s.autoRunning = false
			s.autoCancel = nil
			s.autoNextAt = time.Time{}
		}
		payload := s.autoSenderPayloadLocked()
		s.autoMu.Unlock()
		s.broadcast(webEvent{Type: "auto", Data: payload})
	}()

	accountIndex := 0
	messageIndex := 0
	for {
		select {
		case <-ctx.Done():
			s.addChat(chatPayload{Text: "auto-sender stopped", System: true})
			return
		default:
		}

		accountName := accounts[accountIndex%len(accounts)]
		message := messages[messageIndex%len(messages)]
		if err := s.autoSendOnce(ctx, writers, accountName, message); err != nil {
			s.autoMu.Lock()
			if s.autoID == autoID {
				s.autoError = fmt.Sprintf("%s: %s", accountName, err)
			}
			payload := s.autoSenderPayloadLocked()
			s.autoMu.Unlock()
			s.broadcast(webEvent{Type: "auto", Data: payload})
			s.addChat(chatPayload{
				Text:   "auto-sender error: " + accountName + ": " + err.Error(),
				System: true,
			})
		} else {
			s.autoMu.Lock()
			if s.autoID == autoID {
				s.autoSent++
				s.autoLastAccount = accountName
				s.autoLastMessage = message
				s.autoError = ""
			}
			payload := s.autoSenderPayloadLocked()
			s.autoMu.Unlock()
			s.broadcast(webEvent{Type: "auto", Data: payload})
		}

		accountIndex++
		messageIndex++
		delay := randomAutoDelay(minSeconds, maxSeconds)
		nextAt := time.Now().Add(delay)
		s.autoMu.Lock()
		if s.autoID == autoID {
			s.autoNextAt = nextAt
		}
		payload := s.autoSenderPayloadLocked()
		s.autoMu.Unlock()
		s.broadcast(webEvent{Type: "auto", Data: payload})

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			s.addChat(chatPayload{Text: "auto-sender stopped", System: true})
			return
		case <-timer.C:
		}
	}
}

func (s *webSession) autoSendOnce(ctx context.Context, writers map[string]*twitchIRCConn, accountName, message string) error {
	s.mu.Lock()
	acc, _, err := s.findAccountLocked(accountName)
	channel := s.channel
	s.mu.Unlock()
	if err != nil {
		return err
	}

	key := strings.ToLower(acc.Name)
	writer := writers[key]
	if writer == nil {
		connectCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		resolved, err := s.resolveAccount(connectCtx, acc)
		if err == nil {
			writer, err = newLiveWriter(connectCtx, resolved, channel)
		}
		cancel()
		if err != nil {
			return err
		}
		writers[key] = writer
	}
	if err := writer.Say(channel, message); err != nil {
		writer.Close()
		delete(writers, key)
		return err
	}
	return nil
}

func randomAutoDelay(minSeconds, maxSeconds int) time.Duration {
	if maxSeconds <= minSeconds {
		return time.Duration(minSeconds) * time.Second
	}
	delta := maxSeconds - minSeconds + 1
	return time.Duration(minSeconds+rand.Intn(delta)) * time.Second
}

func (s *webSession) stopAutoSender() {
	s.autoMu.Lock()
	cancel := s.autoCancel
	if cancel != nil {
		cancel()
	}
	s.autoCancel = nil
	s.autoRunning = false
	s.autoNextAt = time.Time{}
	payload := s.autoSenderPayloadLocked()
	s.autoMu.Unlock()
	if cancel != nil {
		s.broadcast(webEvent{Type: "auto", Data: payload})
	}
}

func (s *webSession) autoSenderPayloadWithFile() autoSenderPayload {
	payload := s.autoSenderPayload()
	if lines, err := readAutoSenderLines(autoSenderFileName); err == nil {
		payload.FileLines = len(lines)
	} else if payload.Error == "" {
		payload.Error = err.Error()
	}
	return payload
}

func (s *webSession) autoSenderPayload() autoSenderPayload {
	s.autoMu.Lock()
	defer s.autoMu.Unlock()
	return s.autoSenderPayloadLocked()
}

func (s *webSession) autoSenderPayloadLocked() autoSenderPayload {
	accounts := append([]string(nil), s.autoAccounts...)
	if accounts == nil {
		accounts = []string{}
	}
	payload := autoSenderPayload{
		Running:     s.autoRunning,
		Accounts:    accounts,
		MinSeconds:  s.autoMinSeconds,
		MaxSeconds:  s.autoMaxSeconds,
		File:        autoSenderFileName,
		FileLines:   s.autoFileLines,
		Sent:        s.autoSent,
		LastAccount: s.autoLastAccount,
		LastMessage: s.autoLastMessage,
		Error:       s.autoError,
	}
	if !s.autoNextAt.IsZero() {
		payload.NextAt = s.autoNextAt.Format(time.RFC3339)
	}
	return payload
}

func (s *webSession) statePayload() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	accounts := make([]accountPayload, 0, len(s.cfg.Accounts))
	for _, acc := range s.cfg.Accounts {
		login := acc.Login
		userID := acc.UserID
		if strings.EqualFold(acc.Name, s.current.Name) {
			login = s.current.Login
			userID = s.current.UserID
		}
		accounts = append(accounts, accountPayload{
			Name:        acc.Name,
			Login:       login,
			UserID:      userID,
			HasProxy:    acc.SOCKS5Proxy != "",
			SOCKS5Proxy: acc.SOCKS5Proxy,
			Active:      s.current.Name != "" && strings.EqualFold(acc.Name, s.current.Name),
		})
	}
	return map[string]any{
		"channel":     s.channel,
		"current":     s.current.Name,
		"login":       s.current.Login,
		"accounts":    accounts,
		"presets":     s.cfg.Presets,
		"auto_sender": s.autoSenderPayload(),
	}
}

func (s *webSession) findAccountLocked(name string) (Account, int, error) {
	name = strings.TrimSpace(name)
	if name == "" && len(s.cfg.Accounts) > 0 {
		return s.cfg.Accounts[0], 0, nil
	}
	for i, acc := range s.cfg.Accounts {
		if strings.EqualFold(acc.Name, name) || strings.EqualFold(acc.Login, name) {
			return acc, i, nil
		}
	}
	return Account{}, -1, fmt.Errorf("account %q not found", name)
}

func (s *webSession) addChat(msg chatPayload) {
	if msg.Time == "" {
		msg.Time = time.Now().Format("15:04:05")
	}
	s.mu.Lock()
	if msg.ID != "" {
		for _, existing := range s.chatMessages {
			if existing.ID == msg.ID {
				s.mu.Unlock()
				return
			}
		}
	}
	s.chatMessages = append(s.chatMessages, msg)
	if len(s.chatMessages) > 300 {
		s.chatMessages = s.chatMessages[len(s.chatMessages)-300:]
	}
	s.mu.Unlock()
	s.broadcast(webEvent{Type: "chat", Data: msg})
}

func (s *webSession) history() []chatPayload {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]chatPayload, len(s.chatMessages))
	copy(out, s.chatMessages)
	return out
}

func (s *webSession) broadcast(event webEvent) {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()
	for ch := range s.clients {
		select {
		case ch <- event:
		default:
			log.Printf("dropping web event for slow client: %s", event.Type)
		}
	}
}

func (s *webSession) close() {
	s.stopAutoSender()
	s.mu.Lock()
	writer := s.writer
	reader := s.reader
	s.mu.Unlock()
	if writer != nil {
		writer.Close()
	}
	if reader != nil {
		reader.Close()
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func rewardState(reward api.CustomReward) string {
	if !reward.Enabled {
		return "disabled"
	}
	if reward.Paused {
		return "paused"
	}
	return "enabled"
}
