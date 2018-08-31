package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/Devs-On-Discord/DoDdy/embed"
	"github.com/anmitsu/go-shlex"
	"github.com/bwmarrin/discordgo"
	bolt "go.etcd.io/bbolt"
)

var prefixes = map[string]string{}

func getPrefix(guildID string, username string) string {
	if prefix, ok := prefixes[guildID]; ok && prefixes[guildID] != "" {
		return prefix
	}
	return fmt.Sprintf("@%s ", username)
}

const errColor = 0xb30000
const okColor = 0x00b300

type deletionTarget struct {
	commandID    string
	answerID     string
	channelID    string
	deletionTime time.Time
}

// The message deletion channel is used to schedule messages for deletion, without having to keep a goroutine alive
var deletionChannel chan deletionTarget

func handleMessageCreate(s *discordgo.Session, h *discordgo.MessageCreate) {
	if h.Author.ID == s.State.User.ID {
		return
	}

	if len(h.Content) == 0 {
		return
	}

	input := h.Content

	channel, err := s.Channel(h.ChannelID)
	if err != nil {
		return
	}

	if h.Content[:1] == "<" && len(h.Content) >= 2 { // Called by mention
		nickSpacing := 0
		if h.Content[2:3] == "!" {
			nickSpacing = 1
		}
		if len(h.Content) >= len(s.State.User.ID)+3+nickSpacing && h.Content[2+nickSpacing:len(s.State.User.ID)+2+nickSpacing] == s.State.User.ID {
			input = input[len(s.State.User.ID)+3+nickSpacing : len(input)]
		} else {
			return
		}
	} else if prefix, ok := prefixes[channel.GuildID]; ok && h.Content[:1] == prefix { // Called by prefix
		input = input[1:len(input)]
	} else {
		return
	}

	command, err := shlex.Split(input, true)
	var message = "Unknown error, please contact admins and report this."
	var color = errColor
	if err != nil {
		message = "Could not parse command: " + err.Error()
	} else {
		fmt.Println(command)
		if len(command) == 0 || command[0] == "help" {
			dm, err := s.UserChannelCreate(h.Author.ID)
			if err != nil {
				message = "Unable to initiate DM with the user."
			} else {
				_, err := s.ChannelMessageSendEmbed(dm.ID, embed.NewEmbed().SetTitle("Pretend this is the help string").MessageEmbed)
				if err != nil {
					message = "Can't DM help, please allow DMs from this server."
				} else {
					s.ChannelMessageDelete(h.ChannelID, h.ID)
					return
				}
			}
		} else {
			message = fmt.Sprintf("Command not recognized: %s", command[0])
		}
	}

	if command[0] == "prefix" {
		if len(command) < 2 {
			message = fmt.Sprintf("Invalid syntax: correct syntax looks like `%sprefix /`", getPrefix(channel.GuildID, s.State.User.Username))
		} else {
			if len(command[1]) > 1 {
				message = "Invalid prefix: the prefix should only be one character."
				if len(command[1]) == 4 && command[1] == "none" {
					prefixes[channel.GuildID] = command[1]
					db.Update(func(tx *bolt.Tx) error {
						nodeBucket, err := tx.CreateBucketIfNotExists([]byte("Nodes"))
						if err != nil {
							return err
						}
						guildBucket, err := nodeBucket.CreateBucketIfNotExists([]byte(channel.GuildID))
						if err != nil {
							return err
						}
						if guildBucket.Delete([]byte("Prefix")) != nil {
							return err
						}
						return nil
					})
					if err != nil {
						message = "Database error: " + err.Error()
					} else {
						message = fmt.Sprintf("Prefix deleted")
						color = okColor
					}
				}
			} else {
				if strings.ContainsAny(command[1], "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890< ") {
					message = "Invalid prefix: the prefix should not be a letter (a-z, A-Z), nor a number (0-9), nor the character '<' or a whitespace"
				} else {
					prefixes[channel.GuildID] = command[1]
					db.Update(func(tx *bolt.Tx) error {
						nodeBucket, err := tx.CreateBucketIfNotExists([]byte("Nodes"))
						if err != nil {
							return err
						}
						guildBucket, err := nodeBucket.CreateBucketIfNotExists([]byte(channel.GuildID))
						if err != nil {
							return err
						}
						if guildBucket.Put([]byte("Prefix"), []byte(command[1])) != nil {
							return err
						}
						return nil
					})
					if err != nil {
						message = "Database error: " + err.Error()
					} else {
						message = fmt.Sprintf("Prefix set to '%s'", command[1])
						color = okColor
					}
				}
			}
		}
	}

	answer, _ := s.ChannelMessageSendEmbed(h.ChannelID, embed.NewEmbed().SetColor(color).SetTitle(message).SetFooter("Deletion in 10 seconds").MessageEmbed)
	deletionChannel <- deletionTarget{
		commandID:    h.ID,
		answerID:     answer.ID,
		channelID:    h.ChannelID,
		deletionTime: time.Now().Add(10 * time.Second),
	}

}

func deleter(input chan deletionTarget, s *discordgo.Session) {
	for {
		select {
		case x, ok := <-input:

			if time.Now().After(x.deletionTime) {
				s.ChannelMessageDelete(x.channelID, x.commandID)
				s.ChannelMessageDelete(x.channelID, x.answerID)
			} else {
				if ok {
					input <- x
					time.Sleep(10 * time.Millisecond)
				}
			}
		}
	}
}
