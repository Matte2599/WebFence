package credentials

import (
	"context"
	"errors"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"
)

const secretService = "org.freedesktop.secrets"
const secretRoot dbus.ObjectPath = "/org/freedesktop/secrets"
const secretInterface = "org.freedesktop.Secret."
const nativeTimeout = 3 * time.Second

type linuxBackend struct{ service string }

func nativeBackend(service string) backend { return linuxBackend{service} }

// Accept one local Unix endpoint only: no TCP, autolaunch, or fallback addresses.
func unixBusAddress(address string) (string, error) {
	if !strings.HasPrefix(address, "unix:") || strings.Contains(address, ";") {
		return "", ErrUnavailable
	}
	var result string
	seen := map[string]bool{}
	for _, field := range strings.Split(strings.TrimPrefix(address, "unix:"), ",") {
		key, value, ok := strings.Cut(field, "=")
		if !ok || seen[key] {
			return "", ErrUnavailable
		}
		seen[key] = true
		decoded, err := dbus.UnescapeBusAddressValue(value)
		if err != nil || decoded == "" || strings.ContainsRune(decoded, 0) {
			return "", ErrUnavailable
		}
		switch key {
		case "path":
			if result != "" || !strings.HasPrefix(decoded, "/") {
				return "", ErrUnavailable
			}
			result = decoded
		case "abstract":
			if result != "" {
				return "", ErrUnavailable
			}
			result = "\x00" + decoded
		case "guid": // Informational bus ID, never a destination.
			if len(decoded) != 32 || strings.Trim(decoded, "0123456789abcdefABCDEF") != "" {
				return "", ErrUnavailable
			}
		default:
			return "", ErrUnavailable
		}
	}
	if result == "" {
		return "", ErrUnavailable
	}
	return result, nil
}

// Own the socket and connection so cancellation interrupts both authentication
// and method calls. Only EXTERNAL authentication over the Unix socket is used.
func connectBus(ctx context.Context, address string) (*dbus.Conn, error) {
	endpoint, err := unixBusAddress(address)
	if err != nil {
		return nil, err
	}
	socket, err := (&net.Dialer{}).DialContext(ctx, "unix", endpoint)
	if err != nil {
		return nil, err
	}
	conn, err := dbus.NewConn(socket, dbus.WithContext(ctx))
	if err != nil {
		socket.Close()
		return nil, err
	}
	if err = conn.Auth([]dbus.Auth{dbus.AuthExternal(strconv.Itoa(os.Geteuid()))}); err == nil {
		err = conn.Hello()
	}
	if err != nil {
		conn.Close()
		return nil, err
	}
	return conn, nil
}

type secretValue struct {
	Session     dbus.ObjectPath
	Parameters  []byte
	Value       []byte
	ContentType string
}
type secretClient struct {
	conn       *dbus.Conn
	owner      string
	ctx        context.Context
	collection dbus.ObjectPath
}

func (c secretClient) object(path dbus.ObjectPath) dbus.BusObject {
	return c.conn.Object(c.owner, path)
}
func (c secretClient) locked(path dbus.ObjectPath, iface string) error {
	var value dbus.Variant
	err := c.object(path).CallWithContext(c.ctx, "org.freedesktop.DBus.Properties.Get", 0, secretInterface+iface, "Locked").Store(&value)
	if err != nil {
		return err
	}
	locked, ok := value.Value().(bool)
	if !ok {
		return ErrUnavailable
	}
	if locked {
		return ErrLocked
	}
	return nil
}
func linuxError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	var native dbus.Error
	if errors.As(err, &native) && native.Name == secretInterface+"Error.IsLocked" {
		return ErrLocked
	}
	return sanitize(err)
}
func (b linuxBackend) run(ctx context.Context, op int, id string, value []byte) (result []byte, err error) {
	ctx, cancel := context.WithTimeout(ctx, nativeTimeout)
	defer cancel()
	defer func() { err = linuxError(ctx, err) }()
	conn, err := connectBus(ctx, os.Getenv("DBUS_SESSION_BUS_ADDRESS"))
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	c := secretClient{conn: conn, ctx: ctx}
	// Do not auto-start, unlock, create a collection, or invoke a prompt.
	err = conn.BusObject().CallWithContext(ctx, "org.freedesktop.DBus.GetNameOwner", 0, secretService).Store(&c.owner)
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(c.owner, ":") {
		return nil, ErrUnavailable
	}
	err = c.object(secretRoot).CallWithContext(ctx, secretInterface+"Service.ReadAlias", 0, "default").Store(&c.collection)
	if err != nil {
		return nil, err
	}
	if c.collection == "/" || !c.collection.IsValid() {
		return nil, ErrUnavailable
	}
	if err = c.locked(c.collection, "Collection"); err != nil {
		return nil, err
	}
	attributes := map[string]string{"service": b.service, "account": id}
	var items []dbus.ObjectPath
	err = c.object(c.collection).CallWithContext(ctx, secretInterface+"Collection.SearchItems", 0, attributes).Store(&items)
	if err != nil {
		return nil, err
	}
	if len(items) > 1 {
		return nil, ErrConflict
	}
	if len(items) == 1 {
		if !items[0].IsValid() || items[0] == "/" {
			return nil, ErrUnavailable
		}
		if err = c.locked(items[0], "Item"); err != nil {
			return nil, err
		}
	} else if op != 1 {
		return nil, ErrNotFound
	}
	if op == 2 {
		var prompt dbus.ObjectPath
		err = c.object(items[0]).CallWithContext(ctx, secretInterface+"Item.Delete", 0).Store(&prompt)
		if err != nil {
			return nil, err
		}
		if prompt != "/" {
			return nil, ErrLocked
		}
		return nil, nil
	}
	var output dbus.Variant
	var session dbus.ObjectPath
	err = c.object(secretRoot).CallWithContext(ctx, secretInterface+"Service.OpenSession", 0, "plain", dbus.MakeVariant("")).Store(&output, &session)
	if err != nil {
		return nil, err
	}
	if session == "/" || !session.IsValid() || output.Value() != "" {
		return nil, ErrUnavailable
	}
	defer c.object(session).CallWithContext(ctx, secretInterface+"Session.Close", 0)
	if op == 0 {
		var secret secretValue
		err = c.object(items[0]).CallWithContext(ctx, secretInterface+"Item.GetSecret", 0, session).Store(&secret)
		if err != nil {
			clear(secret.Value)
			return nil, err
		}
		if secret.Session != session || len(secret.Parameters) != 0 || secret.ContentType != "application/octet-stream" {
			clear(secret.Value)
			return nil, ErrUnavailable
		}
		return secret.Value, nil
	}
	secret := secretValue{session, []byte{}, value, "application/octet-stream"}
	if len(items) == 1 {
		err = c.object(items[0]).CallWithContext(ctx, secretInterface+"Item.SetSecret", 0, secret).Err
		return nil, err
	}
	props := map[string]dbus.Variant{
		secretInterface + "Item.Label":      dbus.MakeVariant("WebFence credential"),
		secretInterface + "Item.Attributes": dbus.MakeVariant(attributes),
	}
	var item, prompt dbus.ObjectPath
	err = c.object(c.collection).CallWithContext(ctx, secretInterface+"Collection.CreateItem", 0, props, secret, true).Store(&item, &prompt)
	if err != nil {
		return nil, err
	}
	if prompt != "/" {
		return nil, ErrLocked
	}
	if item == "/" || !item.IsValid() {
		return nil, ErrUnavailable
	}
	return nil, nil
}
func (b linuxBackend) get(ctx context.Context, id string) ([]byte, error) {
	return b.run(ctx, 0, id, nil)
}
func (b linuxBackend) set(ctx context.Context, id string, value []byte) error {
	_, err := b.run(ctx, 1, id, value)
	return err
}
func (b linuxBackend) delete(ctx context.Context, id string) error {
	_, err := b.run(ctx, 2, id, nil)
	return err
}
