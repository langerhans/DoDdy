package commands

import "github.com/bwmarrin/discordgo"

// Handle is the function signature used by handling commands
type Handle func(session *discordgo.Session, commandMessage *discordgo.MessageCreate, args []string) CommandResultMessage

// Command encapsulates a command name and it's handler
type Command struct {
	Name    string
	Handler Handle
}
