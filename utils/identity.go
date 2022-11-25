package utils

import (
	"bytes"
	"errors"
	"fmt"
	"os"

	"filippo.io/age"
	"filippo.io/age/agessh"
)

func AddOpenSSHIdentities(identities *[]age.Identity) {
	// If they exist and are well-formed, load the default SSH keys. If they are
	// passphrase protected, the passphrase will only be requested if the
	// identity matches a recipient stanza.
	for _, path := range []string{
		os.ExpandEnv("$HOME/.ssh/id_rsa"),
		os.ExpandEnv("$HOME/.ssh/id_ed25519"),
	} {
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		ids, err := ParseSSHIdentity(path, content)
		if err != nil {
			// If the key is explicitly requested, this error will be caught
			// below, otherwise ignore it silently.
			continue
		}
		*identities = append(*identities, ids...)
	}
}

func IdentitiesToRecipients(ids []age.Identity) ([]age.Recipient, error) {
	var recipients []age.Recipient
	for _, id := range ids {
		switch id := id.(type) {
		case *age.X25519Identity:
			recipients = append(recipients, id.Recipient())
		case *agessh.RSAIdentity:
			recipients = append(recipients, id.Recipient())
		case *agessh.Ed25519Identity:
			recipients = append(recipients, id.Recipient())
		case *agessh.EncryptedSSHIdentity:
			recipients = append(recipients, id.Recipient())
		case *EncryptedIdentity:
			r, err := id.Recipients()
			if err != nil {
				return nil, err
			}
			recipients = append(recipients, r...)
		default:
			return nil, fmt.Errorf("unexpected identity type: %T", id)
		}
	}
	return recipients, nil
}

var _ age.Identity = (*EncryptedIdentity)(nil)

type EncryptedIdentity struct {
	Contents       []byte
	Passphrase     func() (string, error)
	NoMatchWarning func()

	identities []age.Identity
}

func (i *EncryptedIdentity) Recipients() ([]age.Recipient, error) {
	if i.identities == nil {
		if err := i.decrypt(); err != nil {
			return nil, err
		}
	}

	return IdentitiesToRecipients(i.identities)
}

func (i *EncryptedIdentity) Unwrap(stanzas []*age.Stanza) (fileKey []byte, err error) {
	if i.identities == nil {
		if err := i.decrypt(); err != nil {
			return nil, err
		}
	}

	for _, id := range i.identities {
		fileKey, err = id.Unwrap(stanzas)
		if errors.Is(err, age.ErrIncorrectIdentity) {
			continue
		}
		if err != nil {
			return nil, err
		}
		return fileKey, nil
	}
	i.NoMatchWarning()
	return nil, age.ErrIncorrectIdentity
}

func (i *EncryptedIdentity) decrypt() error {
	d, err := age.Decrypt(bytes.NewReader(i.Contents), &LazyScryptIdentity{i.Passphrase})
	if e := new(age.NoIdentityMatchError); errors.As(err, &e) {
		return fmt.Errorf("identity file is encrypted with age but not with a passphrase")
	}
	if err != nil {
		return fmt.Errorf("failed to decrypt identity file: %v", err)
	}
	i.identities, err = age.ParseIdentities(d)
	return err
}

type LazyScryptIdentity struct {
	Passphrase func() (string, error)
}

var _ age.Identity = (*LazyScryptIdentity)(nil)

func (i *LazyScryptIdentity) Unwrap(stanzas []*age.Stanza) (fileKey []byte, err error) {
	for _, s := range stanzas {
		if s.Type == "scrypt" && len(stanzas) != 1 {
			return nil, errors.New("an scrypt recipient must be the only one")
		}
	}
	if len(stanzas) != 1 || stanzas[0].Type != "scrypt" {
		return nil, age.ErrIncorrectIdentity
	}
	pass, err := i.Passphrase()
	if err != nil {
		return nil, fmt.Errorf("could not read passphrase: %v", err)
	}
	ii, err := age.NewScryptIdentity(pass)
	if err != nil {
		return nil, err
	}
	fileKey, err = ii.Unwrap(stanzas)
	if errors.Is(err, age.ErrIncorrectIdentity) {
		// ScryptIdentity returns ErrIncorrectIdentity for an incorrect
		// passphrase, which would lead Decrypt to returning "no identity
		// matched any recipient". That makes sense in the API, where there
		// might be multiple configured ScryptIdentity. Since in cmd/age there
		// can be only one, return a better error message.
		return nil, fmt.Errorf("incorrect passphrase")
	}
	return fileKey, err
}
