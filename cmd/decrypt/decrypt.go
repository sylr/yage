// Copyright 2021 Google LLC
// Copyright 2021 Sylvain Rabot
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file or at
// https://developers.google.com/open-source/licenses/bsd

package decrypt

import (
	"bufio"
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"log"
	"os"

	"filippo.io/age"
	"filippo.io/age/armor"
	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v3"
	"golang.org/x/term"

	"sylr.dev/yage/v2/utils"
	yage "sylr.dev/yaml/age/v3"
)

var (
	outFlag              string
	passFlag             bool
	yamlFlag             bool
	yamlNoTagFlag        bool
	yamlDiscardNoTagFlag bool
	identityFlags        []string

	//go:embed examples.txt
	examples string
)

var DecryptCmd = cobra.Command{
	Use:               "decrypt",
	Aliases:           []string{"d", "dec", "decode"},
	Short:             "Decrypt AGE encrypted data",
	GroupID:           "age",
	SilenceUsage:      true,
	Args:              cobra.MaximumNArgs(1),
	PersistentPreRunE: Validate,
	RunE:              Run,
	Example:           examples,
}

func init() {
	DecryptCmd.PersistentFlags().BoolVarP(&passFlag, "passphrase", "p", false, "Use a passphrase")
	DecryptCmd.PersistentFlags().StringVarP(&outFlag, "output", "o", "", "Output to `FILE` (default stdout)")
	DecryptCmd.PersistentFlags().StringArrayVarP(&identityFlags, "identity", "i", []string{}, "Identity private key for decrypting")
	DecryptCmd.PersistentFlags().BoolVarP(&yamlFlag, "yaml", "y", false, "In-place yaml decrypting")
	DecryptCmd.PersistentFlags().BoolVar(&yamlNoTagFlag, "yaml-notag", false, "Strip !crypto/age tag from output")
	DecryptCmd.PersistentFlags().BoolVar(&yamlDiscardNoTagFlag, "yaml-discard-notag", false, "Do not honour NoTag YAML tag attribute")

	if err := cobra.MarkFlagFilename(DecryptCmd.PersistentFlags(), "identity"); err != nil {
		panic(err)
	}
}

func Validate(_ *cobra.Command, _ []string) error {
	if yamlNoTagFlag && yamlDiscardNoTagFlag {
		return fmt.Errorf("can't use --yaml-notag and --yaml-discard-notag simultaneously.")
	}
	return nil
}

func Run(_ *cobra.Command, args []string) error {
	log.SetFlags(0)

	var in io.Reader = os.Stdin
	var out io.Writer = os.Stdout
	outputName := outFlag
	stdinInUse := false

	var inputName string
	if len(args) > 0 {
		inputName = args[0]
	}

	if inputName != "" && inputName != "-" {
		f, err := os.Open(inputName)
		if err != nil {
			return fmt.Errorf("failed to open input file %q: %w", inputName, err)
		}
		defer f.Close()
		in = f
	} else {
		stdinInUse = true
	}

	if outputName != "" && outputName != "-" {
		if !stdinInUse {
			_, err := os.Stat(inputName)
			if err != nil {
				return fmt.Errorf("failed to open input file %q: %w", inputName, err)
			}

			_, err = os.Stat(outputName)
			if err == nil {
				return fmt.Errorf("output file %q exists", outputName)
			}
		}

		f := utils.NewLazyOpener(outputName, false)
		defer f.Close()
		out = f
	} else if term.IsTerminal(int(os.Stdout.Fd())) {
		if outputName == "-" {
			if in == os.Stdin && term.IsTerminal(int(os.Stdin.Fd())) {
				// If the input comes from a TTY and output will go to a TTY,
				// buffer it up so it doesn't get in the way of typing the input.
				buf := &bytes.Buffer{}
				defer func() { io.Copy(os.Stdout, buf) }() // nolint:errcheck
				out = buf
			}
		}
	}

	if yamlFlag {
		return DecryptYAML(identityFlags, in, out, stdinInUse, yamlNoTagFlag, yamlDiscardNoTagFlag)
	}

	return Decrypt(identityFlags, in, out, stdinInUse)
}

func Decrypt(keys []string, in io.Reader, out io.Writer, stdinInUse bool) error {
	identities := []age.Identity{
		// If there is a scrypt recipient (it will have to be the only one)
		// this identity will be invoked.
		&utils.LazyScryptIdentity{utils.PassphrasePrompt},
	}

	utils.AddOpenSSHIdentities(&identities)

	for _, name := range keys {
		ids, err := utils.ParseIdentitiesFile(name, stdinInUse)
		if err != nil {
			return fmt.Errorf("error reading %q: %w", name, err)
		}
		identities = append(identities, ids...)
	}

	rr := bufio.NewReader(in)
	if start, _ := rr.Peek(len(armor.Header)); string(start) == armor.Header {
		in = armor.NewReader(rr)
	} else {
		in = rr
	}

	r, err := age.Decrypt(in, identities...)
	if err != nil {
		return err
	}

	if _, err := io.Copy(out, r); err != nil {
		return err
	}

	return nil
}

func DecryptYAML(keys []string, in io.Reader, out io.Writer, stdinInUse, noTag bool, discardNoTag bool) error {
	identities := []age.Identity{
		// If there is a scrypt recipient (it will have to be the only one)
		// this identity will be invoked.
		&utils.LazyScryptIdentity{utils.PassphrasePrompt},
	}

	utils.AddOpenSSHIdentities(&identities)

	for _, name := range keys {
		ids, err := utils.ParseIdentitiesFile(name, stdinInUse)
		if err != nil {
			return fmt.Errorf("error reading %q: %v", name, err)
		}
		identities = append(identities, ids...)
	}

	node := yaml.Node{}
	w := yage.Wrapper{
		Value:        &node,
		Identities:   identities,
		ForceNoTag:   noTag,
		DiscardNoTag: discardNoTag,
	}

	decoder := yaml.NewDecoder(in)
	encoder := yaml.NewEncoder(out)
	encoder.SetIndent(2)
	defer encoder.Close()

	for {
		err := decoder.Decode(&w)
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		err = encoder.Encode(&node)
		if err != nil {
			return err
		}
	}

	return nil
}
