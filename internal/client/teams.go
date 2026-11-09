package client

import (
	"context"
	"fmt"
	"net/http"
)

type Team struct {
	ID                     int    `json:"id"`
	Name                   string `json:"name"`
	Description            string `json:"description,omitempty"`
	ScanImagesOnNewServers *bool  `json:"scan_images_on_new_servers,omitempty"`
}

// UpdateCurrentTeamInput is PATCH /team. Only scan_images_on_new_servers is
// sent. Coolify >= v4.7.0. UpdateCurrentTeam skips the PATCH when the
// pointer is nil or the instance is older, because an empty body is rejected
// and the older allow list does not include this field.
type UpdateCurrentTeamInput struct {
	ScanImagesOnNewServers *bool `json:"scan_images_on_new_servers,omitempty"`
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

func (c *Client) ListTeams(ctx context.Context) ([]Team, error) {
	var t []Team
	if err := c.do(ctx, http.MethodGet, "/api/v1/teams", nil, &t); err != nil {
		return nil, fmt.Errorf("listing teams: %w", err)
	}
	return t, nil
}

// UpdateCurrentTeam patches the team bound to the API token.
func (c *Client) UpdateCurrentTeam(ctx context.Context, input UpdateCurrentTeamInput) (*Team, error) {
	if !c.SupportsCoolify47() {
		input.ScanImagesOnNewServers = nil
	}
	if input.ScanImagesOnNewServers == nil {
		return c.GetCurrentTeam(ctx)
	}
	var t Team
	if err := c.do(ctx, http.MethodPatch, "/api/v1/team", input, &t); err != nil {
		return nil, fmt.Errorf("updating current team: %w", err)
	}
	return &t, nil
}

func (c *Client) GetCurrentTeam(ctx context.Context) (*Team, error) {
	var t Team
	if err := c.do(ctx, http.MethodGet, "/api/v1/teams/current", nil, &t); err != nil {
		return nil, fmt.Errorf("getting current team: %w", err)
	}
	return &t, nil
}

func (c *Client) GetCurrentTeamMembers(ctx context.Context) ([]TeamMember, error) {
	var m []TeamMember
	if err := c.do(ctx, http.MethodGet, "/api/v1/teams/current/members", nil, &m); err != nil {
		return nil, fmt.Errorf("getting current team members: %w", err)
	}
	return m, nil
}
