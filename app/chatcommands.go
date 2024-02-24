package app

import (
	"fmt"
	"strings"

	"mleku.dev/git/nostr/event"
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

- admin : permission to change lower privilege levels on user accounts - only owners can change admins

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
	if prefix != "" {
		// this is an error print
		prefix = strings.TrimSpace(prefix) + "\n\n"
		replyString += cmd.Help
		for i := range Commands {
			split := strings.Split(Commands[i].Help, "\n")
			replyString += split[0]
			replyString += "\n ➞ " + split[2]
			replyString += "\n\n"
		}
	} else {
		// this is a direct invocation
		if len(args) == 1 {
			replyString += cmd.Help
			for i := range Commands {
				if Commands[i].Name == "help" {
					continue
				}
				split := strings.Split(Commands[i].Help, "\n")
				replyString += split[0]
				replyString += "\n ➞ " + split[2]
				replyString += "\n\n"
			}
		} else if len(args) >= 2 {
			// pretty much all commands are one level deep so second item is the
			// help item:
			for i := range Commands {
				if Commands[i].Name == args[1] {
					if Commands[i].Name == "help" {
						for j := range Commands {
							if Commands[j].Name == "help" {
								replyString += cmd.Help
								continue
							}
							split := strings.Split(Commands[j].Help, "\n")
							replyString += split[0]
							replyString += "\n ➞ " + split[2]
							replyString += "\n\n"
						}
					} else {
						replyString = Commands[i].Help
					}
				}
			}
			if replyString == "" {
				replyString = fmt.Sprintf(`command '%s' unknown

type 'help' to see a list of valid commands`, strings.Join(args[1:], " "))
			}
		}
	}
	reply = MakeReply(ev, fmt.Sprintf("%s%s", prefix, replyString))
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
