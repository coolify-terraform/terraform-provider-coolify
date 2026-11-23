package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// DeleteBackupsOptions selects the delete_backups query on application,
// database, and service deletes. Coolify's default is false. The flag is
// omitted unless DeleteBackups is true, so older Coolify versions keep the
// current DELETE URL.
type DeleteBackupsOptions struct {
	DeleteBackups bool
}

func (c *Client) deleteWithBackups(ctx context.Context, kind, prefix, uuid string, opts DeleteBackupsOptions) error {
	if err := c.rejectDeleteBackups(kind, uuid, opts); err != nil {
		return err
	}
	path := prefix + url.PathEscape(uuid)
	if opts.DeleteBackups {
		path += "?delete_backups=true"
	}
	if err := c.do(ctx, http.MethodDelete, path, nil, nil); err != nil {
		return fmt.Errorf("deleting %s %s: %w", kind, uuid, err)
	}
	return nil
}

func (c *Client) rejectDeleteBackups(kind, uuid string, opts DeleteBackupsOptions) error {
	if !opts.DeleteBackups || c.SupportsResourceDeleteFlags() {
		return nil
	}
	return fmt.Errorf("deleting %s %s: delete_backups requires Coolify >= v%s (this instance is %s)", kind, uuid, minResourceDeleteFlagsVersion, reportedCoolifyVersion(c))
}
