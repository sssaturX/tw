package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/adeithe/go-twitch/irc"
)

type liveSession struct {
	cfg      Config
	channel  string
	resolved map[string]resolvedAccount

	current resolvedAccount
	writer  *twitchIRCConn
	outMu   sync.Mutex
}

func liveCommand(args []string) error {
	fs := flag.NewFlagSet("live", flag.ContinueOnError)
	configPath := fs.String("config", defaultConfigPath, "path to accounts config")
	accountName := fs.String("account", "", "initial account name/login")
	channel := fs.String("channel", "", "target channel login")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *channel == "" {
		return errors.New("--channel is required")
	}

	cfg, err := loadConfig(*configPath)
	if err != nil {
		return err
	}
	accounts, err := selectAccounts(cfg, *accountName, false)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	session := &liveSession{
		cfg:      cfg,
		channel:  normalizeLogin(*channel),
		resolved: map[string]resolvedAccount{},
	}

	if err := session.switchAccount(ctx, accounts[0].Name); err != nil {
		return err
	}
	defer session.close()

	reader, err := newTwitchIRCConn(Account{})
	if err != nil {
		return err
	}
	reader.OnMessage(func(msg irc.ChatMessage) {
		displayName := msg.Sender.DisplayName
		if displayName == "" {
			displayName = msg.Sender.Username
		}
		session.outputf("\n[%s] %s: %s\n%s", msg.CreatedAt.Format("15:04:05"), displayName, msg.Text, session.prompt())
	})
	reader.OnServerNotice(func(notice irc.ServerNotice) {
		session.outputf("\n[notice] %s\n%s", notice.Message, session.prompt())
	})
	defer reader.Close()

	if err := reader.Connect(ctx); err != nil {
		return fmt.Errorf("chat reader connect failed: %w", err)
	}
	if err := reader.Join(session.channel); err != nil {
		return fmt.Errorf("chat reader join failed: %w", err)
	}

	session.outputf("Live chat #%s. Type /help for commands.\n%s", session.channel, session.prompt())
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			session.outputf("%s", session.prompt())
			continue
		}

		if strings.HasPrefix(line, "/") {
			quit, err := session.handleCommand(ctx, line)
			if err != nil {
				session.outputf("error: %v\n", err)
			}
			if quit {
				return nil
			}
			session.outputf("%s", session.prompt())
			continue
		}

		if err := session.writer.Say(session.channel, line); err != nil {
			session.outputf("send failed from %s: %v\n", session.current.Login, err)
		}
		session.outputf("%s", session.prompt())
	}
	return scanner.Err()
}

func (s *liveSession) switchAccount(ctx context.Context, name string) error {
	acc, err := s.findAccount(name)
	if err != nil {
		return err
	}

	resolved, ok := s.resolved[strings.ToLower(acc.Name)]
	if !ok {
		ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
		defer cancel()

		resolved, err = validateAccount(ctx, acc)
		if err != nil {
			return err
		}
		s.resolved[strings.ToLower(acc.Name)] = resolved
		if resolved.Login != "" {
			s.resolved[strings.ToLower(resolved.Login)] = resolved
		}
	}

	nextWriter, err := newLiveWriter(ctx, resolved, s.channel)
	if err != nil {
		return err
	}

	oldWriter := s.writer
	s.writer = nextWriter
	s.current = resolved
	if oldWriter != nil {
		oldWriter.Close()
	}

	s.outputf("active account: %s (%s)\n", resolved.Name, resolved.Login)
	return nil
}

func newLiveWriter(ctx context.Context, acc resolvedAccount, channel string) (*twitchIRCConn, error) {
	conn, err := newTwitchIRCConn(acc.Account)
	if err != nil {
		return nil, err
	}
	if err := conn.Connect(ctx); err != nil {
		return nil, err
	}
	if err := conn.Join(channel); err != nil {
		conn.Close()
		return nil, err
	}

	select {
	case <-ctx.Done():
		conn.Close()
		return nil, ctx.Err()
	case <-time.After(750 * time.Millisecond):
	}
	return conn, nil
}

func (s *liveSession) handleCommand(ctx context.Context, line string) (bool, error) {
	fields := strings.Fields(line)
	command := strings.TrimPrefix(strings.ToLower(fields[0]), "/")

	switch command {
	case "q", "quit", "exit":
		return true, nil
	case "help":
		s.outputf(`Commands:
  /accounts          list configured accounts
  /account NAME      switch active account
  /whoami            show active account
  /emotes [kind]     list first 20 emotes; kind: global, channel, user
  /rewards           list broadcaster custom rewards for this channel
  /quit              exit live mode
`)
	case "accounts":
		for _, acc := range s.cfg.Accounts {
			marker := " "
			if strings.EqualFold(acc.Name, s.current.Name) || strings.EqualFold(acc.Login, s.current.Login) {
				marker = "*"
			}
			login := acc.Login
			if login == "" {
				login = "not validated yet"
			}
			s.outputf("%s %s (%s)\n", marker, acc.Name, login)
		}
	case "account":
		if len(fields) < 2 {
			return false, errors.New("usage: /account NAME")
		}
		return false, s.switchAccount(ctx, fields[1])
	case "whoami":
		s.outputf("%s (%s, user_id=%s)\n", s.current.Name, s.current.Login, s.current.UserID)
	case "emotes":
		kind := "channel"
		if len(fields) >= 2 {
			kind = strings.ToLower(fields[1])
		}
		ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
		defer cancel()

		emotes, err := listEmotes(ctx, s.current, kind, s.channel)
		if err != nil {
			return false, err
		}
		limit := 20
		if len(emotes) < limit {
			limit = len(emotes)
		}
		for i := 0; i < limit; i++ {
			s.outputf("%s\t%s\t%s\n", emotes[i].Name, emotes[i].ID, emotes[i].EmoteType)
		}
		if len(emotes) > limit {
			s.outputf("... %d more\n", len(emotes)-limit)
		}
	case "rewards":
		ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
		defer cancel()

		rewards, err := listCustomRewards(ctx, s.current, s.channel, false)
		if err != nil {
			return false, err
		}
		for _, reward := range rewards {
			state := "enabled"
			if reward.Paused {
				state = "paused"
			}
			if !reward.Enabled {
				state = "disabled"
			}
			s.outputf("%s\t%d\t%s\t%s\n", reward.RewardID, reward.Cost, state, reward.Title)
		}
	default:
		return false, fmt.Errorf("unknown live command %q", fields[0])
	}

	return false, nil
}

func (s *liveSession) findAccount(name string) (Account, error) {
	name = strings.TrimSpace(name)
	if name == "" && len(s.cfg.Accounts) > 0 {
		return s.cfg.Accounts[0], nil
	}
	for _, acc := range s.cfg.Accounts {
		if strings.EqualFold(acc.Name, name) || strings.EqualFold(acc.Login, name) {
			return acc, nil
		}
	}
	return Account{}, fmt.Errorf("account %q not found", name)
}

func (s *liveSession) prompt() string {
	login := s.current.Login
	if login == "" {
		login = s.current.Name
	}
	return fmt.Sprintf("[%s -> #%s] > ", login, s.channel)
}

func (s *liveSession) outputf(format string, args ...any) {
	s.outMu.Lock()
	defer s.outMu.Unlock()
	fmt.Printf(format, args...)
}

func (s *liveSession) close() {
	if s.writer != nil {
		s.writer.Close()
	}
}
