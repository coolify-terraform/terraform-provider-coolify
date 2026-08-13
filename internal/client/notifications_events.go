package client

import (
	"fmt"
	"reflect"
)

// NotificationEvents is the 14 shared Coolify notification event flags
// (channel-agnostic). Field names must match the exported event fields on
// every *NotificationSettings and Update*NotificationInput struct.
type NotificationEvents struct {
	DeploymentSuccess    bool
	DeploymentFailure    bool
	StatusChange         bool
	BackupSuccess        bool
	BackupFailure        bool
	ScheduledTaskSuccess bool
	ScheduledTaskFailure bool
	DockerCleanupSuccess bool
	DockerCleanupFailure bool
	ServerDiskUsage      bool
	ServerReachable      bool
	ServerUnreachable    bool
	ServerPatch          bool
	TraefikOutdated      bool
}

// NotificationEventUpdate is the PATCH form of NotificationEvents (nil = omit).
type NotificationEventUpdate struct {
	DeploymentSuccess    *bool
	DeploymentFailure    *bool
	StatusChange         *bool
	BackupSuccess        *bool
	BackupFailure        *bool
	ScheduledTaskSuccess *bool
	ScheduledTaskFailure *bool
	DockerCleanupSuccess *bool
	DockerCleanupFailure *bool
	ServerDiskUsage      *bool
	ServerReachable      *bool
	ServerUnreachable    *bool
	ServerPatch          *bool
	TraefikOutdated      *bool
}

// ApplyEventUpdate copies shared event pointers onto a channel Update* input.
// dst must be a pointer to UpdateDiscordNotificationInput or the other five
// channel update types. Extra destination fields are left unchanged.
func ApplyEventUpdate(dst any, ev NotificationEventUpdate) error {
	return copyFieldsByName(dst, ev, false)
}

// EventsFrom copies shared event bools from a channel *NotificationSettings.
func EventsFrom(src any) (NotificationEvents, error) {
	var out NotificationEvents
	if err := copyFieldsByName(&out, src, true); err != nil {
		return NotificationEvents{}, err
	}
	return out, nil
}

// copyFieldsByName copies exported fields from src onto dst by name.
// When requireSrc is true, every exported dst field must exist on src.
func copyFieldsByName(dst, src any, requireSrc bool) error {
	d, err := structElem(dst)
	if err != nil {
		return fmt.Errorf("dst: %w", err)
	}
	s, err := structElem(src)
	if err != nil {
		return fmt.Errorf("src: %w", err)
	}
	if requireSrc {
		return copyDestFields(d, s)
	}
	return copySrcFields(d, s)
}

func copyDestFields(d, s reflect.Value) error {
	dt := d.Type()
	for i := 0; i < dt.NumField(); i++ {
		df := dt.Field(i)
		if !df.IsExported() {
			continue
		}
		sf := s.FieldByName(df.Name)
		if err := assignField(d.Field(i), sf, df.Name, s.Type()); err != nil {
			return err
		}
	}
	return nil
}

func copySrcFields(d, s reflect.Value) error {
	st := s.Type()
	for i := 0; i < st.NumField(); i++ {
		sf := st.Field(i)
		if !sf.IsExported() {
			continue
		}
		df := d.FieldByName(sf.Name)
		if err := assignField(df, s.Field(i), sf.Name, d.Type()); err != nil {
			return err
		}
	}
	return nil
}

func assignField(dst, src reflect.Value, name string, missingOn reflect.Type) error {
	if !src.IsValid() || !dst.IsValid() || !dst.CanSet() {
		return fmt.Errorf("%s missing field %s", missingOn, name)
	}
	if !src.Type().AssignableTo(dst.Type()) {
		return fmt.Errorf("field %s: cannot assign %s to %s", name, src.Type(), dst.Type())
	}
	dst.Set(src)
	return nil
}

func structElem(v any) (reflect.Value, error) {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return reflect.Value{}, fmt.Errorf("nil value")
	}
	if k := rv.Kind(); k == reflect.Pointer {
		if rv.IsNil() {
			return reflect.Value{}, fmt.Errorf("nil pointer")
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return reflect.Value{}, fmt.Errorf("want struct, got %s", rv.Kind())
	}
	return rv, nil
}
