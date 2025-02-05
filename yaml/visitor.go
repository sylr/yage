package yaml

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"filippo.io/age"
	"filippo.io/age/armor"
	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
)

const (
	// YAMLTag tag that is used to identify data to encrypt/decrypt
	YAMLTag       = "!crypto/age"
	YAMLTagPrefix = "!crypto/age:"
)

var (
	ErrUnknownAttribute          = fmt.Errorf("unknown attribute")
	ErrMoreThanOneStyleAttribute = fmt.Errorf("can't use more than one style attribute")
	ErrUpstreamAgeError          = fmt.Errorf("age")
	ErrUnsupportedValueType      = fmt.Errorf("unsupported Value type")
)

var _ ast.Visitor = &Decoder{}

type Decoder struct {
	// Identities that will be used to try decrypting encrypted Value.
	Identities []age.Identity

	// NoDecrypt inscruts the Unmarshaler to leave encrypted data encrypted.
	// This is useful when you want to Marshal new un-encrytped data in a
	// document already containing encrypted data.
	NoDecrypt bool

	errors []error
}

func (v *Decoder) Errors() []error {
	return v.errors
}

func (v *Decoder) Visit(node ast.Node) ast.Visitor {
	if v.NoDecrypt {
		return nil
	}

	n, ok := node.(*ast.TagNode)
	if !ok {
		return v
	}

	if n.Start.Value != YAMLTag && !strings.HasPrefix(n.Start.Value, YAMLTagPrefix) {
		return v
	}

	stringReader := strings.NewReader(n.Value.GetToken().Next.Value)
	armoredReader := armor.NewReader(stringReader)
	decryptedReader, err := age.Decrypt(armoredReader, v.Identities...)
	if err != nil {
		v.errors = append(v.errors, err)
		return v
	}
	buf2 := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf2, decryptedReader); err != nil {
		v.errors = append(v.errors, err)
		return v
	}
	if n2, err := yaml.ValueToNode(buf2.String()); err != nil {
		v.errors = append(v.errors, err)
		return v
	} else {
		if err := n2.SetComment(n.Value.GetComment()); err != nil {
			v.errors = append(v.errors, err)
			return v
		}
		n2.SetPath(n.Value.GetPath())
		n.Value = n2
	}

	return v
}

var _ ast.Visitor = &Encoder{}

type Encoder struct {
	// Recipients that will be used for encrypting un-encrypted Value.
	Recipients []age.Recipient

	// DiscardNoTag instructs the Unmarshaler to not honour the NoTag
	// `!crypto/age` tag attribute. This is useful when re-keying data.
	DiscardNoTag bool

	// ForceNoTag strip the `!crypto/age` tags from the Marshaler output.
	ForceNoTag bool

	// NoEncrypt inscruts the Unmarshaler to leave encrypted data encrypted.
	// This is useful when you want to Marshal new un-encrytped data in a
	// document already containing encrypted data.
	NoEncrypt bool

	errors []error
}

func (v *Encoder) Errors() []error {
	return v.errors
}

func (v *Encoder) Visit(node ast.Node) ast.Visitor {
	n, ok := node.(*ast.TagNode)
	if !ok {
		return v
	}

	notag := false

	if n.Start.Value != YAMLTag {
		if strings.HasPrefix(n.Start.Value, YAMLTagPrefix) {
			attrStr := n.Start.Value[len(YAMLTagPrefix):]
			attrs := strings.Split(attrStr, ",")
			for _, attr := range attrs {
				lower := strings.ToLower(attr)
				switch lower {
				case "doublequoted", "singlequoted", "literal", "folded", "flow":
					// Discard old style attributes
				case "notag":
					notag = true
				default:
					v.errors = append(v.errors, fmt.Errorf("%w: %s", ErrUnknownAttribute, attrStr))
				}
			}
		} else {
			return v
		}
	}

	if isArmoredAgeFile(n.Value.String()) {
		return v
	}

	if !v.NoEncrypt {
		buf := bytes.NewBuffer(nil)
		armorWriter := armor.NewWriter(buf)
		encryptWriter, err := age.Encrypt(armorWriter, v.Recipients...)
		if err != nil {
			v.errors = append(v.errors, err)
			return v
		}

		_, err = io.WriteString(encryptWriter, n.Value.String())
		if err != nil {
			v.errors = append(v.errors, err)
			return v
		}

		encryptWriter.Close()
		armorWriter.Close()

		if n2, err := yaml.ValueToNode(buf.String()); err != nil {
			v.errors = append(v.errors, err)
			return v
		} else {
			if err := n2.SetComment(n.Value.GetComment()); err != nil {
				v.errors = append(v.errors, err)
				return v
			}
			n2.SetPath(n.Value.GetPath())
			n.Value = n2
		}
	}

	if v.ForceNoTag || (!v.DiscardNoTag && notag) {
		n.Start = n.Value.GetToken()
	}

	return v
}

func isArmoredAgeFile(data string) bool {
	trimmed := strings.TrimSpace(data)
	return strings.HasPrefix(trimmed, armor.Header) && strings.HasSuffix(trimmed, armor.Footer)
}
