package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

type Tag struct {
	UUID      string `json:"uuid"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

type CreateTagInput struct {
	Name string `json:"name"`
}

type UpdateTagInput struct {
	Name string `json:"name"`
}

type AttachTagsInput struct {
	TagName  string   `json:"tag_name,omitempty"`
	TagNames []string `json:"tag_names,omitempty"`
}

func (c *Client) ListTags(ctx context.Context) ([]Tag, error) {
	var r []Tag
	if err := c.doCachedList(ctx, "/api/v1/tags", &r); err != nil {
		return nil, fmt.Errorf("listing tags: %w", err)
	}
	return r, nil
}

func (c *Client) GetTag(ctx context.Context, uuid string) (*Tag, error) {
	tags, err := c.ListTags(ctx)
	if err != nil {
		return nil, err
	}
	for i := range tags {
		if tags[i].UUID == uuid {
			return &tags[i], nil
		}
	}
	return nil, &NotFoundError{Message: fmt.Sprintf("tag %s not found", uuid)}
}

func (c *Client) CreateTag(ctx context.Context, input CreateTagInput) (*Tag, error) {
	var r Tag
	if err := c.doWithStatus(ctx, http.MethodPost, "/api/v1/tags", input, &r, http.StatusCreated); err != nil {
		return nil, fmt.Errorf("creating tag: %w", err)
	}
	c.listCache.invalidate("/api/v1/tags")
	return &r, nil
}

func (c *Client) UpdateTag(ctx context.Context, uuid string, input UpdateTagInput) (*Tag, error) {
	var r Tag
	if err := c.do(ctx, http.MethodPatch, fmt.Sprintf("/api/v1/tags/%s", url.PathEscape(uuid)), input, &r); err != nil {
		return nil, fmt.Errorf("updating tag %s: %w", uuid, err)
	}
	c.listCache.invalidate("/api/v1/tags")
	return &r, nil
}

func (c *Client) DeleteTag(ctx context.Context, uuid string) error {
	if err := c.do(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/tags/%s", url.PathEscape(uuid)), nil, nil); err != nil {
		return fmt.Errorf("deleting tag %s: %w", uuid, err)
	}
	c.listCache.invalidate("/api/v1/tags")
	return nil
}

func taggablePrefix(resourceType string) (string, error) {
	switch resourceType {
	case "application":
		return "applications", nil
	case "database":
		return "databases", nil
	case "service":
		return "services", nil
	default:
		return "", fmt.Errorf("unsupported tag resource type %q", resourceType)
	}
}

func (c *Client) ListResourceTags(ctx context.Context, resourceType, resourceUUID string) ([]Tag, error) {
	prefix, err := taggablePrefix(resourceType)
	if err != nil {
		return nil, err
	}
	var r []Tag
	path := fmt.Sprintf("/api/v1/%s/%s/tags", prefix, url.PathEscape(resourceUUID))
	if err := c.do(ctx, http.MethodGet, path, nil, &r); err != nil {
		return nil, fmt.Errorf("listing tags on %s %s: %w", resourceType, resourceUUID, err)
	}
	return r, nil
}

func (c *Client) AttachResourceTag(ctx context.Context, resourceType, resourceUUID, tagName string) error {
	prefix, err := taggablePrefix(resourceType)
	if err != nil {
		return err
	}
	path := fmt.Sprintf("/api/v1/%s/%s/tags", prefix, url.PathEscape(resourceUUID))
	if err := c.doWithStatus(ctx, http.MethodPost, path, AttachTagsInput{TagName: tagName}, nil, http.StatusCreated); err != nil {
		return fmt.Errorf("attaching tag %q to %s %s: %w", tagName, resourceType, resourceUUID, err)
	}
	return nil
}

func (c *Client) DetachResourceTag(ctx context.Context, resourceType, resourceUUID, tagUUID string) error {
	prefix, err := taggablePrefix(resourceType)
	if err != nil {
		return err
	}
	path := fmt.Sprintf("/api/v1/%s/%s/tags/%s", prefix, url.PathEscape(resourceUUID), url.PathEscape(tagUUID))
	if err := c.do(ctx, http.MethodDelete, path, nil, nil); err != nil {
		return fmt.Errorf("detaching tag %s from %s %s: %w", tagUUID, resourceType, resourceUUID, err)
	}
	return nil
}
