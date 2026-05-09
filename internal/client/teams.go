package client

import (
	"context"
	"fmt"
	"net/http"
)

type Team struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}
type TeamMember struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (c *Client) GetTeam(ctx context.Context, id int) (*Team, error) {
	var t Team
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/teams/%d", id), nil, &t); err != nil {
		return nil, fmt.Errorf("getting team %d: %w", id, err)
	}
	return &t, nil
}
func (c *Client) ListTeamMembers(ctx context.Context, id int) ([]TeamMember, error) {
	var m []TeamMember
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/teams/%d/members", id), nil, &m); err != nil {
		return nil, fmt.Errorf("listing team %d members: %w", id, err)
	}
	return m, nil
}
