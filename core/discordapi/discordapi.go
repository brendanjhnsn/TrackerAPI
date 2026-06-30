package discordapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

const DefaultBaseURL = "https://discord.com/api"

type guildMember struct {
	User  *discordUser `json:"user"`
	Roles []string     `json:"roles"`
}

type discordUser struct {
	ID string `json:"id"`
}

// ListMembersWithRole returns the Discord user IDs of every guild member holding roleID.
// baseURL is exposed for testing; pass discordapi.DefaultBaseURL in production.
// pageSize controls how many members are fetched per request (use 1000 in production).
func ListMembersWithRole(ctx context.Context, client *http.Client, baseURL, botToken, guildID, roleID string, pageSize int) ([]string, error) {
	var result []string
	after := ""
	for {
		url := fmt.Sprintf("%s/guilds/%s/members?limit=%d", baseURL, guildID, pageSize)
		if after != "" {
			url += "&after=" + after
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bot "+botToken)
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		var members []guildMember
		err = json.NewDecoder(resp.Body).Decode(&members)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}
		for _, mem := range members {
			if mem.User == nil {
				continue
			}
			if hasRole(mem.Roles, roleID) {
				result = append(result, mem.User.ID)
			}
		}
		if len(members) < pageSize {
			break
		}
		after = members[len(members)-1].User.ID
	}
	return result, nil
}

func hasRole(roles []string, roleID string) bool {
	for _, r := range roles {
		if r == roleID {
			return true
		}
	}
	return false
}
