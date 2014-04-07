package ssh

import (
	"encoding/json"
	"os/user"
)

var keyTypeHandler map[string]KeyTypeHandler
var permissionHandlers map[string]PermissionHandler
var authorizedKeysHandlers []AuthorizedKeysHandler

type KeyTypeHandler interface {
	CreateKey(key json.RawMessage) (KeyLocator, error)
}
type KeyLocator interface {
	PathToKey() string
	NameForKey() string
}

type PermissionHandler interface {
	CreatePermission(locator KeyLocator, permission json.RawMessage) error
}

type AuthorizedKeysHandler interface {
	MatchesUser(*user.User) bool
	GenerateAuthorizedKeysFile(*user.User) error
}

// Register a function to generate authorized keys for a system user.
func AddKeyTypeHandler(id string, handler KeyTypeHandler) {
	if keyTypeHandler == nil {
		keyTypeHandler = make(map[string]KeyTypeHandler)
	}
	keyTypeHandler[id] = handler
}

func KeyTypeHandlerFor(id string) (KeyTypeHandler, bool) {
	handler, ok := keyTypeHandler[id]
	return handler, ok
}

// Register a function to generate authorized keys for a system user.
func AddPermissionHandler(id string, handler PermissionHandler) {
	if permissionHandlers == nil {
		permissionHandlers = make(map[string]PermissionHandler)
	}
	permissionHandlers[id] = handler
}

func PermissionHandlerFor(id string) (PermissionHandler, bool) {
	handler, ok := permissionHandlers[id]
	return handler, ok
}

// Register a function to generate authorized keys for a system user.
func AddAuthorizedKeyGenerationType(handler AuthorizedKeysHandler) {
	authorizedKeysHandlers = append(authorizedKeysHandlers, handler)
}

type SimpleKeyLocator struct {
	Path string
	Name string
}

func (l *SimpleKeyLocator) PathToKey() string {
	return l.Path
}
func (l *SimpleKeyLocator) NameForKey() string {
	return l.Name
}
