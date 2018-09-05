package botcommands

import (
	"github.com/Devs-On-Discord/DoDdy/guilds"
	"github.com/bwmarrin/discordgo"
)

type botCommandIdentifier struct {
	guilds *guilds.Guilds
}

func (i botCommandIdentifier) Identify(s *discordgo.Session, m *discordgo.MessageCreate) bool {
	botID := s.State.User.ID
	if m.Author.ID == botID {
		return false
	}
	if len(m.Content) == 0 {
		return false
	}
	for _, mention := range m.Mentions {
		if mention.ID == botID {
			m.Content = m.Content[len(mention.ID)+3:] //<@{botID}>
			return true
		}
	}
	if len(m.Content) > 1 {
		if guild, err := i.guilds.Guild(m.GuildID); err == nil {
			if guild.Prefix == m.Content[:1] {
				m.Content = m.Content[1:]
				return true
			}
		}
	}
	return false
}
