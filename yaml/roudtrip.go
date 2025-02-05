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

var _ ast.Visitor = &RoundTrip{}

type RoundTrip struct {
	// Identities that will be used to try decrypting encrypted Value.
	Identities []age.Identity
	// Recipients that will be used for encrypting un-encrypted Value.
	Recipients []age.Recipient
	// NoDecrypt instructs the Unmarshaler to leave encrypted data encrypted.
	// This is useful when you want to Marshal new un-encrytped data in a
	// document already containing encrypted data.
	NoDecrypt bool
	// DiscardNoTag instructs the Marshaller to not honour the NoTag
	// `!crypto/age` tag attribute. This is useful when re-keying data.
	DiscardNoTag bool
	// ForceNoTag strip the `!crypto/age` tags from the Marshaler output.
	ForceNoTag bool

	errors []error
}

func (v *RoundTrip) Errors() []error {
	return v.errors
}

func (v *RoundTrip) Visit(node ast.Node) ast.Visitor {
	n, ok := node.(*ast.TagNode)
	if !ok {
		return v
	}

	var notag bool
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

	v.decode(n, notag)
	v.encode(n, notag)
	return nil
}

func (v *RoundTrip) decode(n *ast.TagNode, notag bool) {
	if v.NoDecrypt {
		return
	}

	if n.Start.Value != YAMLTag && !strings.HasPrefix(n.Start.Value, YAMLTagPrefix) {
		return
	}

	stringReader := strings.NewReader(n.Value.GetToken().Next.Value)
	armoredReader := armor.NewReader(stringReader)
	decryptedReader, err := age.Decrypt(armoredReader, v.Identities...)
	if err != nil {
		v.errors = append(v.errors, err)
		return
	}
	buf2 := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf2, decryptedReader); err != nil {
		v.errors = append(v.errors, err)
		return
	}
	if n2, err := yaml.ValueToNode(buf2.String()); err != nil {
		v.errors = append(v.errors, err)
		return
	} else {
		if err := n2.SetComment(n.Value.GetComment()); err != nil {
			v.errors = append(v.errors, err)
			return
		}
		n2.SetPath(n.Value.GetPath())
		n.Value = n2
	}
}

func (v *RoundTrip) encode(n *ast.TagNode, notag bool) {
	// No recipients, nothing to encrypt
	if len(v.Recipients) == 0 {
		return
	}
	// Data is already encrypted
	if isArmoredAgeFile(n.Value.String()) {
		return
	}

	buf := bytes.NewBuffer(nil)
	armorWriter := armor.NewWriter(buf)
	encryptWriter, err := age.Encrypt(armorWriter, v.Recipients...)
	if err != nil {
		v.errors = append(v.errors, err)
		return
	}

	_, err = io.WriteString(encryptWriter, n.Value.String())
	if err != nil {
		v.errors = append(v.errors, err)
		return
	}

	encryptWriter.Close()
	armorWriter.Close()

	if n2, err := yaml.ValueToNode(buf.String()); err != nil {
		v.errors = append(v.errors, err)
		return
	} else {
		if err := n2.SetComment(n.Value.GetComment()); err != nil {
			v.errors = append(v.errors, err)
			return
		}
		n2.SetPath(n.Value.GetPath())
		n.Value = n2
	}

	if v.ForceNoTag || (!v.DiscardNoTag && notag) {
		n.Start = n.Value.GetToken()
	}
}
