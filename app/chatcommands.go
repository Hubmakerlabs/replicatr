package app

import (
	"fmt"
	"strings"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
)

var Commands []*Command

func (rl *Relay) Init() {
	Commands = []*Command{
		{
			Name: "help",
			Help: `help [command name]

shows help for a command

this relay chat bot understands the following commands:
`,
			Func: help,
		},
		{
			Name: "set",
			Help: `set <pubkey> [admin|writer|reader|none|denied]

sets the permission for access by the user with <pubkey> to the relay

- admin : being able to change lower privilege levels on user accounts - only owners can change admins

- writer : permission to request events and publish events to the relay

- reader : permission to request events from the relay

- none : no permission, this is effectively the same as denied on auth-required relay

- denied : that this user will not have any requests or events accepted by the relay
`,
			Func: set,
		},
		{
			Name: "list",
			Help: `list [admin|writer|reader|none|denied]

returns the list of pubkeys, optionally from a given privilege level

only owner and admin users can use this command
`,
			Func: list,
		},
	}
}

func help(rl *Relay, prefix string, ev *event.T, cmd *Command, args ...string) (reply *event.T, err error) {
	var replyString string
	switch {
	case len(args) < 2:
		replyString += cmd.Help
		for i := range Commands {
			replyString += "\n"
			split := strings.Split(Commands[i].Help, "\n")
			replyString += split[0]
			replyString += "\n ➞ " + split[2]
			replyString += "\n"
		}
		replyString += "\n"
		reply = MakeReply(ev, fmt.Sprintf(replyString))
	case len(args) == 2 && args[1] == "help":
		if prefix != "" {
			replyString = prefix + "\n\n"
		}
		replyString += cmd.Help
		for i := range Commands {
			replyString += "\n"
			split := strings.Split(Commands[i].Help, "\n")
			replyString += split[0]
			replyString += "\n ➞ " + split[2]
			replyString += "\n"
		}
		replyString += "\n"
		reply = MakeReply(ev, fmt.Sprintf(replyString))
	default:
		for i := range Commands {
			if Commands[i].Name == args[1] {
				replyString = strings.TrimSpace(Commands[i].Help)
				reply = MakeReply(ev, fmt.Sprintf(replyString))
				return
			}
		}
	}
	return
}

func set(rl *Relay, prefix string, ev *event.T, cmd *Command, args ...string) (reply *event.T, err error) {
	reply = MakeReply(ev, fmt.Sprintf("set not implemented yet\n args: %v\nevent: %s\nprefix: %s", args, ev.ToObject().String(), prefix))
	return
}

func list(rl *Relay, prefix string, ev *event.T, cmd *Command, args ...string) (reply *event.T, err error) {
	reply = MakeReply(ev, fmt.Sprintf("list not implemented yet\n args: %v\nevent: %s\nprefix: %s", args, ev.ToObject().String(), prefix))
	return
}
