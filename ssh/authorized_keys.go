package ssh

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	key "github.com/openshift/geard/pkg/ssh-public-key"
	"github.com/openshift/geard/utils"
)

func init() {
	AddKeyTypeHandler("authorized_keys", &authorizedKeyType{})
}

type authorizedKeyType struct{}

func (t authorizedKeyType) CreateKey(raw json.RawMessage) (KeyLocator, error) {
	pk, _, _, _, ok := key.ParseAuthorizedKey([]byte(raw))

	if !ok {
		return nil, errors.New("Unable to parse the provided key")
	}

	value := key.MarshalAuthorizedKey(pk)
	fingerprint := KeyFingerprint(pk)
	path := fingerprint.PublicKeyPathFor()

	if err := utils.AtomicWriteToContentPath(path, 0664, value); err != nil {
		return nil, err
	}
	return &SimpleKeyLocator{path, fingerprint.ToShortName()}, nil
}

func KeyFingerprint(key key.PublicKey) utils.Fingerprint {
	bytes := sha256.Sum256(key.Marshal())
	return utils.Fingerprint(bytes[:])
}
