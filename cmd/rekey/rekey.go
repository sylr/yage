// Copyright 2021 Google LLC
// Copyright 2021 Sylvain Rabot
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file or at
// https://developers.google.com/open-source/licenses/bsd

package rekey

import (
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"filippo.io/age"
	"filippo.io/age/armor"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"sylr.dev/yage/cmd/decrypt"
	"sylr.dev/yage/utils"
	yage "sylr.dev/yaml/age/v3"
	"sylr.dev/yaml/v3"
)

var (
	outFlag                string
	armorFlag              bool
	passFlag               bool
	yamlFlag               bool
	yamlDiscardNotagFlag   bool
	recipientFlags         []string
	recipientFileFlags     []string
	recipientIdentityFlags []string
	identityFlags          []string

	//go:embed examples.txt
	examples string
)

var RekeyCmd = cobra.Command{
	Use:               "rekey",
	Short:             "Re-key data with new set of recipients",
	GroupID:           "age",
	SilenceUsage:      true,
	Args:              cobra.MaximumNArgs(1),
	PersistentPreRunE: Validate,
	RunE:              Run,
	Example:           examples,
}

func init() {
	RekeyCmd.PersistentFlags().BoolVarP(&passFlag, "passphrase", "p", false, "Use a passphrase")
	RekeyCmd.PersistentFlags().StringVarP(&outFlag, "output", "o", "", "Output to `FILE` (default stdout)")
	RekeyCmd.PersistentFlags().BoolVarP(&armorFlag, "armor", "a", false, "Generate an armored file")
	RekeyCmd.PersistentFlags().StringArrayVarP(&recipientFlags, "recipient", "r", []string{}, "Recipient public keys")
	RekeyCmd.PersistentFlags().StringArrayVarP(&recipientFileFlags, "recipient-file", "R", []string{}, "Recipient public key file")
	RekeyCmd.PersistentFlags().StringArrayVar(&recipientIdentityFlags, "recipient-identity", []string{}, "Recipient identity private key (used to derive public key which will be added as recipient)")
	RekeyCmd.PersistentFlags().StringArrayVarP(&identityFlags, "identity", "i", []string{}, "Identity private key (used for decrypting)")
	RekeyCmd.PersistentFlags().BoolVarP(&yamlFlag, "yaml", "y", false, "In-place yaml encrypting/decrypting")
	RekeyCmd.PersistentFlags().BoolVar(&yamlDiscardNotagFlag, "yaml-discard-notag", false, "Do not honour NoTag YAML tag attribute")

	RekeyCmd.InitDefaultCompletionCmd()

	if err := cobra.MarkFlagFilename(RekeyCmd.PersistentFlags(), "identity"); err != nil {
		panic(err)
	}
	if err := cobra.MarkFlagFilename(RekeyCmd.PersistentFlags(), "recipient"); err != nil {
		panic(err)
	}
	if err := cobra.MarkFlagFilename(RekeyCmd.PersistentFlags(), "recipient-file"); err != nil {
		panic(err)
	}
	if err := cobra.MarkFlagFilename(RekeyCmd.PersistentFlags(), "recipient-identity"); err != nil {
		panic(err)
	}
}

func Validate(_ *cobra.Command, _ []string) error {
	if len(recipientFlags)+len(recipientFileFlags)+len(recipientIdentityFlags) == 0 && !passFlag {
		return fmt.Errorf("missing recipients.\n" +
			"Did you forget to specify -r/--recipient, -R/--recipient-file or -p/--passphrase?")
	}
	if len(recipientFlags) > 0 && passFlag {
		//lint:ignore ST1005 error is displayed by the CLI
		return fmt.Errorf("-p/--passphrase can't be combined with -r/--recipient.")
	}
	if len(recipientFileFlags) > 0 && passFlag {
		//lint:ignore ST1005 error is displayed by the CLI
		return fmt.Errorf("-p/--passphrase can't be combined with -R/--recipient-file.")
	}
	if len(recipientIdentityFlags) > 0 && passFlag {
		//lint:ignore ST1005 error is displayed by the CLI
		return fmt.Errorf("-p/--passphrase can't be combined with -R/--recipient-identity.")
	}
	if yamlFlag {
		armorFlag = true
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
		if outputName != "-" {
			if !armorFlag {
				// If the output wouldn't be armored, refuse to send binary to
				// the terminal unless explicitly requested with "-o -".
				//lint:ignore ST1005 error is displayed by the CLI
				return fmt.Errorf("refusing to output binary to the term.\n" +
					`Did you mean to use -a/--armor? Force with "-o -".`)
			}
		}
		if in == os.Stdin && term.IsTerminal(int(os.Stdin.Fd())) {
			// If the input comes from a TTY and output will go to a TTY,
			// buffer it up so it doesn't get in the way of typing the input.
			buf := &bytes.Buffer{}
			defer func() { io.Copy(os.Stdout, buf) }() // nolint:errcheck
			out = buf
		}
	}

	outbuf := &bytes.Buffer{}
	if yamlFlag {
		if err := decrypt.DecryptYAML(identityFlags, in, outbuf, stdinInUse, false, true); err != nil {
			return err
		}
	} else {
		if err := decrypt.Decrypt(identityFlags, in, outbuf, stdinInUse); err != nil {
			return err
		}
	}

	if passFlag {
		if pass, err := passphrasePromptForEncryption(); err != nil {
			return err
		} else {
			return EncryptPass(pass, outbuf, out, armorFlag, yamlFlag)
		}
	}

	return EncryptKeys(recipientFlags, recipientFileFlags, recipientIdentityFlags, outbuf, out, armorFlag, stdinInUse, yamlFlag)
}

func passphrasePromptForEncryption() (string, error) {
	pass, err := utils.ReadPassphrase("Enter passphrase (leave empty to autogenerate a secure one):")
	if err != nil {
		return "", fmt.Errorf("could not read passphrase: %v", err)
	}
	p := string(pass)
	if p == "" {
		var words []string
		for i := 0; i < 10; i++ {
			words = append(words, utils.RandomWord())
		}
		p = strings.Join(words, "-")
		// TODO: consider printing this to the terminal, instead of stderr.
		fmt.Fprintf(os.Stderr, "Using the autogenerated passphrase %q.\n", p)
	} else {
		confirm, err := utils.ReadPassphrase("Confirm passphrase:")
		if err != nil {
			return "", fmt.Errorf("could not read passphrase: %v", err)
		}
		if string(confirm) != p {
			return "", fmt.Errorf("passphrases didn't match")
		}
	}
	return p, nil
}

func EncryptKeys(keys, files, identities []string, in io.Reader, out io.Writer, armor bool, stdinInUse, yaml bool) error {
	var recipients []age.Recipient

	for _, arg := range keys {
		r, err := utils.ParseRecipient(arg)
		if err, ok := err.(utils.GitHubRecipientError); ok {
			utils.ErrorWithHint(err.Error(), "instead, use recipient files like",
				"    curl -O https://github.com/"+err.Username()+".keys",
				"    yage -R "+err.Username()+".keys")
		}
		if err != nil {
			return err
		}
		recipients = append(recipients, r)
	}

	for _, name := range files {
		recs, err := utils.ParseRecipientsFile(name, stdinInUse)
		if err != nil {
			return fmt.Errorf("failed to parse recipient file %q: %w", name, err)
		}
		recipients = append(recipients, recs...)
	}

	for _, name := range identities {
		ids, err := utils.ParseIdentitiesFile(name, stdinInUse)
		if err != nil {
			return fmt.Errorf("failed to read %q: %w", name, err)
		}
		r, err := utils.IdentitiesToRecipients(ids)
		if err != nil {
			utils.Errorf("internal error processing %q: %w", name, err)
		}
		recipients = append(recipients, r...)
	}

	if yaml {
		return EncryptYAML(recipients, in, out)
	}

	return Encrypt(recipients, in, out, armor)
}

func EncryptPass(pass string, in io.Reader, out io.Writer, armor bool, yaml bool) error {
	r, err := age.NewScryptRecipient(pass)
	if err != nil {
		return err
	}

	if yaml {
		return EncryptYAML([]age.Recipient{r}, in, out)
	}

	return Encrypt([]age.Recipient{r}, in, out, armor)
}

func Encrypt(recipients []age.Recipient, in io.Reader, out io.Writer, withArmor bool) error {
	if withArmor {
		a := armor.NewWriter(out)
		defer func() {
			if err := a.Close(); err != nil {
				log.Fatal("%w", err)
			}
		}()
		out = a
	}

	w, err := age.Encrypt(out, recipients...)
	if err != nil {
		return err
	}

	if _, err := io.Copy(w, in); err != nil {
		return err
	}

	if err := w.Close(); err != nil {
		return err
	}

	return nil
}

func EncryptYAML(recipients []age.Recipient, in io.Reader, out io.Writer) error {
	node := yaml.Node{}
	w := yage.Wrapper{Value: &node, Recipients: recipients, NoDecrypt: true}

	decoder := yaml.NewDecoder(in)
	encoder := yaml.NewEncoder(out)
	encoder.SetIndent(2)
	defer encoder.Close()

	for {
		err := decoder.Decode(&w)
		if err == io.EOF {
			break
		} else if err != nil {
			return fmt.Errorf("yaml decoding failed: %w", err)
		}

		err = encoder.Encode(&w)
		if err != nil {
			return fmt.Errorf("yaml encoding failed: %w", err)
		}
	}

	return nil
}
