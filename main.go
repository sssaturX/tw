package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

const defaultConfigPath = "accounts.json"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	// If no arguments, launch web UI with default settings
	if len(args) == 0 {
		return webCommand([]string{"--channel", "general"})
	}

	switch args[0] {
	case "accounts":
		return accountsCommand(args[1:])
	case "send":
		return sendCommand(args[1:])
	case "live":
		return liveCommand(args[1:])
	case "web":
		return webCommand(args[1:])
	case "emotes":
		return emotesCommand(args[1:])
	case "rewards":
		return rewardsCommand(args[1:])
	case "points":
		return pointsCommand(args[1:])
	case "help", "-h", "--help":
		printUsage()
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func accountsCommand(args []string) error {
	fs := flag.NewFlagSet("accounts", flag.ContinueOnError)
	configPath := fs.String("config", defaultConfigPath, "path to accounts config")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := loadConfig(*configPath)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	for _, acc := range cfg.Accounts {
		resolved, err := validateAccount(ctx, acc)
		if err != nil {
			fmt.Printf("%s: invalid: %v\n", acc.Name, err)
			continue
		}

		chatReady := "missing chat:edit"
		if hasScope(resolved.Validation.Scopes, "chat:edit") || hasScope(resolved.Validation.Scopes, "user:write:chat") {
			chatReady = "chat ok"
		}
		fmt.Printf("%s: login=%s user_id=%s client_id=%s expires_in=%ds scopes=%s (%s)\n",
			resolved.Name,
			resolved.Login,
			resolved.UserID,
			resolved.ClientID,
			resolved.Validation.ExpiresIn,
			strings.Join(resolved.Validation.Scopes, ","),
			chatReady,
		)
	}
	return nil
}

func sendCommand(args []string) error {
	fs := flag.NewFlagSet("send", flag.ContinueOnError)
	configPath := fs.String("config", defaultConfigPath, "path to accounts config")
	accountName := fs.String("account", "", "account name/login")
	all := fs.Bool("all", false, "send from all accounts in config")
	channel := fs.String("channel", "", "target channel login")
	message := fs.String("message", "", "chat message or emote code")
	delay := fs.Duration("delay", 1500*time.Millisecond, "delay between accounts when --all is used")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *channel == "" {
		return errors.New("--channel is required")
	}
	if strings.TrimSpace(*message) == "" {
		return errors.New("--message is required")
	}

	cfg, err := loadConfig(*configPath)
	if err != nil {
		return err
	}
	accounts, err := selectAccounts(cfg, *accountName, *all)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(len(accounts))*30*time.Second)
	defer cancel()

	for i, acc := range accounts {
		resolved, err := validateAccount(ctx, acc)
		if err != nil {
			return err
		}
		if err := sendChatMessage(ctx, resolved, *channel, *message); err != nil {
			return fmt.Errorf("%s send failed: %w", resolved.Name, err)
		}
		fmt.Printf("%s -> #%s: sent\n", resolved.Login, normalizeLogin(*channel))

		if i < len(accounts)-1 && *delay > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(*delay):
			}
		}
	}
	return nil
}

func emotesCommand(args []string) error {
	fs := flag.NewFlagSet("emotes", flag.ContinueOnError)
	configPath := fs.String("config", defaultConfigPath, "path to accounts config")
	accountName := fs.String("account", "", "account name/login")
	kind := fs.String("kind", "global", "global, channel, or user")
	channel := fs.String("channel", "", "channel login for channel/user emotes")
	limit := fs.Int("limit", 50, "max rows to print")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := loadConfig(*configPath)
	if err != nil {
		return err
	}
	accounts, err := selectAccounts(cfg, *accountName, false)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	resolved, err := validateAccount(ctx, accounts[0])
	if err != nil {
		return err
	}
	emotes, err := listEmotes(ctx, resolved, strings.ToLower(*kind), *channel)
	if err != nil {
		return err
	}

	for i, emote := range emotes {
		if *limit > 0 && i >= *limit {
			fmt.Printf("... %d more\n", len(emotes)-i)
			break
		}
		fmt.Printf("%s\t%s\t%s\n", emote.Name, emote.ID, emote.EmoteType)
	}
	return nil
}

func rewardsCommand(args []string) error {
	fs := flag.NewFlagSet("rewards", flag.ContinueOnError)
	configPath := fs.String("config", defaultConfigPath, "path to accounts config")
	accountName := fs.String("account", "", "broadcaster account name/login")
	channel := fs.String("channel", "", "broadcaster login; defaults to selected account")
	onlyManageable := fs.Bool("manageable", false, "only rewards manageable by this app")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := loadConfig(*configPath)
	if err != nil {
		return err
	}
	accounts, err := selectAccounts(cfg, *accountName, false)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	resolved, err := validateAccount(ctx, accounts[0])
	if err != nil {
		return err
	}
	rewards, err := listCustomRewards(ctx, resolved, *channel, *onlyManageable)
	if err != nil {
		return err
	}
	for _, reward := range rewards {
		state := "enabled"
		if reward.Paused {
			state = "paused"
		}
		if !reward.Enabled {
			state = "disabled"
		}
		fmt.Printf("%s\t%d\t%s\t%s\n", reward.RewardID, reward.Cost, state, reward.Title)
	}
	return nil
}

func pointsCommand(args []string) error {
	if len(args) == 0 {
		return errors.New("points subcommand is required: balance or redeem")
	}

	switch args[0] {
	case "balance", "redeem":
		return errors.New("Twitch does not provide an official API for viewer Channel Points balance or viewer-side reward redemption; this tool intentionally does not call private/undocumented endpoints")
	default:
		return fmt.Errorf("unknown points subcommand %q", args[0])
	}
}

func printUsage() {
	fmt.Print(`twitch-multi-tool

Commands:
  accounts   Validate configured OAuth tokens and print account info
  send       Send a chat message or Twitch emote code from one/all accounts
  live       Read chat, write messages, and switch accounts interactively
  web        Start local browser UI for chat and account switching
  emotes     List global, channel, or user emotes
  rewards    List official broadcaster custom rewards
  points     Explains unsupported viewer balance/redeem API

Examples:
  twitch-multi-tool.exe                          (launches web UI)
  twitch-multi-tool.exe accounts
  twitch-multi-tool.exe web --channel somechannel
  twitch-multi-tool.exe live --account bot1 --channel somechannel
  twitch-multi-tool.exe send --account bot1 --channel somechannel --message "Kappa"
  twitch-multi-tool.exe send --all --channel somechannel --message "hello" --delay 2s
  twitch-multi-tool.exe emotes --account bot1 --kind channel --channel somechannel
  twitch-multi-tool.exe rewards --account broadcaster
`)
}
