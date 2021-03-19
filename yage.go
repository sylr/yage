// Copyright 2019 Google LLC
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file or at
// https://developers.google.com/open-source/licenses/bsd

package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	_log "log"
	"os"
	"runtime"
	"runtime/debug"
	"strings"

	"filippo.io/age"
	"filippo.io/age/armor"
	flag "github.com/spf13/pflag"
	"golang.org/x/term"
	yage "sylr.dev/yaml/age/v3"
	"sylr.dev/yaml/v3"
)

type multiFlag []string

func (f *multiFlag) String() string { return fmt.Sprint(*f) }

func (f *multiFlag) Set(value string) error {
	*f = append(*f, value)
	return nil
}

func (f *multiFlag) Type() string {
	return "multiFlag"
}

func (f *multiFlag) Append(value string) error {
	*f = append(*f, value)
	return nil
}

const usage = `Usage:
    yage (-r RECIPIENT | -R PATH)... [--armor] [-o OUTPUT] [INPUT]
    yage --passphrase [--armor] [-o OUTPUT] [INPUT]
    yage --decrypt [-i PATH]... [-o OUTPUT] [INPUT]

Options:
    -o, --output OUTPUT         Write the result to the file at path OUTPUT.
    -a, --armor                 Encrypt to a PEM encoded format.
    -p, --passphrase            Encrypt with a passphrase.
    -r, --recipient RECIPIENT   Encrypt to the specified RECIPIENT. Can be repeated.
    -R, --recipients-file PATH  Encrypt to recipients listed at PATH. Can be repeated.
    -d, --decrypt               Decrypt the input to the output.
    -i, --identity PATH         Use the identity file at PATH. Can be repeated.
        --version
    -y, --yaml                  Treat input as YAML and perform in-place encryption / decryption.
        --yaml-discard-notag    Does not honour NoTag attribute when decrypting (useful for re-keying).
        --rekey                 Decrypt the input and encrypt it with the given recipients.
                                In re-keying mode the input and output can be the same file.
                                In YAML mode it implies --yaml-discard-notag.

INPUT defaults to standard input, and OUTPUT defaults to standard output.

RECIPIENT can be an age public key generated by age-keygen ("age1...")
or an SSH public key ("ssh-ed25519 AAAA...", "ssh-rsa AAAA...").

Recipient files contain one or more recipients, one per line. Empty lines
and lines starting with "#" are ignored as comments. "-" may be used to
read recipients from standard input.

Identity files contain one or more secret keys ("AGE-SECRET-KEY-1..."),
one per line, or an SSH key. Empty lines and lines starting with "#" are
ignored as comments. Multiple key files can be provided, and any unused ones
will be ignored. "-" may be used to read identities from standard input.

Example:
    # Generate age key pair
    $ age-keygen -o key.txt
    Public key: age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p

    # Tar folder and encrypt it with yage
    $ tar cvz ~/data | yage -r age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p > data.tar.gz.age
    $ yage --decrypt -i key.txt -o data.tar.gz data.tar.gz.age

    # Encrypt YAML keys in place tagged with !crypto/age
    $ yage -r age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p -y config.yaml > config.yaml.age

    # Decrypt YAML file encrypted with yage
    $ yage --decrypt -i key.txt --yaml config.yaml.age

    # Re-key age encrypted YAML
    $ yage --rekey --yaml -i key.txt -R ~/.ssh/id_ed25519.pub -R ~/.ssh/id_rsa.pub -o config.yaml.age config.yaml.age
`

// Version can be set at link time to override debug.BuildInfo.Main.Version,
// which is "(devel)" when building from within the module. See
// golang.org/issue/29814 and golang.org/issue/29228.
var Version string

func main() {
	_log.SetFlags(0)
	flag.Usage = func() { fmt.Fprintf(os.Stderr, "%s\n", usage) }

	if len(os.Args) == 1 {
		flag.Usage()
		os.Exit(1)
	}

	var (
		outFlag                        string
		decryptFlag, armorFlag         bool
		passFlag, versionFlag          bool
		yamlFlag, yamlDiscardNotagFlag bool
		rekeyFlag                      bool
		recipientFlags, identityFlags  multiFlag
		recipientsFileFlags            multiFlag
	)

	flag.BoolVar(&versionFlag, "version", false, "print the version")
	flag.BoolVarP(&decryptFlag, "decrypt", "d", false, "decrypt the input")
	flag.BoolVar(&rekeyFlag, "rekey", false, "rekey the input")
	flag.BoolVarP(&passFlag, "passphrase", "p", false, "use a passphrase")
	flag.StringVarP(&outFlag, "output", "o", "", "output to `FILE` (default stdout)")
	flag.BoolVar(&armorFlag, "a", false, "generate an armored file")
	flag.BoolVar(&armorFlag, "armor", false, "generate an armored file")
	flag.VarP(&recipientFlags, "recipient", "r", "recipient (can be repeated)")
	flag.VarP(&recipientsFileFlags, "recipients-file", "R", "recipients file (can be repeated)")
	flag.VarP(&identityFlags, "identity", "i", "identity (can be repeated)")
	flag.BoolVarP(&yamlFlag, "yaml", "y", false, "in-place yaml encrypting/decrypting")
	flag.BoolVar(&yamlDiscardNotagFlag, "yaml-discard-notag", false, "do not honour NoTag YAML tag attribute")
	flag.Parse()

	if versionFlag {
		if Version != "" {
			fmt.Printf("%s (%s)\n", Version, runtime.Version())
			return
		}
		if buildInfo, ok := debug.ReadBuildInfo(); ok {
			fmt.Println(buildInfo.Main.Version)
			return
		}
		fmt.Println("(unknown)")
		return
	}

	if flag.NArg() > 1 {
		logFatalf("Error: too many arguments.\n" +
			"age accepts a single optional argument for the input file.")
	}

	if len(identityFlags) > 0 {
		decryptFlag = true
	}

	switch {
	case rekeyFlag:
	case decryptFlag:
		if armorFlag {
			logFatalf("Error: -a/--armor can't be used with -d/--decrypt.\n" +
				"Note that armored files are detected automatically.")
		}
		if passFlag {
			logFatalf("Error: -p/--passphrase can't be used with -d/--decrypt.\n" +
				"Note that password protected files are detected automatically.")
		}
		if len(recipientFlags) > 0 {
			logFatalf("Error: -r/--recipient can't be used with -d/--decrypt.\n" +
				"Did you mean to use -i/--identity to specify a private key?")
		}
		if len(recipientsFileFlags) > 0 {
			logFatalf("Error: -R/--recipients-file can't be used with -d/--decrypt.\n" +
				"Did you mean to use -i/--identity to specify a private key?")
		}
	default: // encrypt
		if len(identityFlags) > 0 {
			logFatalf("Error: -i/--identity can't be used in encryption mode.\n" +
				"Did you forget to specify -d/--decrypt?")
		}
		if len(recipientFlags) == 0 && len(recipientsFileFlags) == 0 && !passFlag {
			logFatalf("Error: missing recipients.\n" +
				"Did you forget to specify -r/--recipient, -R/--recipients-file or -p/--passphrase?")
		}
		if len(recipientFlags) > 0 && passFlag {
			logFatalf("Error: -p/--passphrase can't be combined with -r/--recipient.")
		}
		if len(recipientsFileFlags) > 0 && passFlag {
			logFatalf("Error: -p/--passphrase can't be combined with -R/--recipients-file.")
		}
		if yamlFlag {
			armorFlag = true
		}
	}

	var in io.Reader = os.Stdin
	var out io.Writer = os.Stdout
	inputName := flag.Arg(0)
	outputName := outFlag

	if inputName != "" && inputName != "-" {
		f, err := os.Open(inputName)
		if err != nil {
			logFatalf("Error: failed to open input file %q: %v", inputName, err)
		}
		defer f.Close()
		in = f
	} else {
		stdinInUse = true
	}

	// --rekey overwrite input file if no output file given
	if rekeyFlag && outputName == "" {
		outputName = inputName
	}

	if outputName != "" && outputName != "-" {
		overwrite := false
		istat, _ := os.Stat(inputName)
		ostat, _ := os.Stat(outputName)
		if rekeyFlag && istat.Name() == ostat.Name() {
			// in rekey mode we allow to overwrite the input file
			overwrite = true
		} else if _, err := os.Stat(outputName); err == nil {
			logFatalf("Error: output file %q exists", outputName)
		}
		f := newLazyOpener(outputName, overwrite)
		defer f.Close()
		out = f
	} else if term.IsTerminal(int(os.Stdout.Fd())) {
		if outputName != "-" {
			if decryptFlag {
				// TODO: buffer the output and check it's printable.
			} else if !armorFlag {
				// If the output wouldn't be armored, refuse to send binary to
				// the terminal unless explicitly requested with "-o -".
				logFatalf("Error: refusing to output binary to the term.\n" +
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

	switch {
	case rekeyFlag:
		outbuf := &bytes.Buffer{}
		if yamlFlag {
			decryptYAML(identityFlags, in, outbuf, true)
		} else {
			decrypt(identityFlags, in, outbuf)
		}
		encryptKeys(recipientFlags, recipientsFileFlags, outbuf, out, armorFlag, yamlFlag)
	case decryptFlag:
		if yamlFlag {
			decryptYAML(identityFlags, in, out, yamlDiscardNotagFlag)
		} else {
			decrypt(identityFlags, in, out)
		}
	case passFlag:
		pass, err := passphrasePromptForEncryption()
		if err != nil {
			logFatalf("Error: %v", err)
		}
		encryptPass(pass, in, out, armorFlag, yamlFlag)
	default:
		encryptKeys(recipientFlags, recipientsFileFlags, in, out, armorFlag, yamlFlag)
	}
}

func passphrasePromptForEncryption() (string, error) {
	fmt.Fprintf(os.Stderr, "Enter passphrase (leave empty to autogenerate a secure one): ")
	pass, err := readPassphrase()
	if err != nil {
		return "", fmt.Errorf("could not read passphrase: %v", err)
	}
	p := string(pass)
	if p == "" {
		var words []string
		for i := 0; i < 10; i++ {
			words = append(words, randomWord())
		}
		p = strings.Join(words, "-")
		fmt.Fprintf(os.Stderr, "Using the autogenerated passphrase %q.\n", p)
	} else {
		fmt.Fprintf(os.Stderr, "Confirm passphrase: ")
		confirm, err := readPassphrase()
		if err != nil {
			return "", fmt.Errorf("could not read passphrase: %v", err)
		}
		if string(confirm) != p {
			return "", fmt.Errorf("passphrases didn't match")
		}
	}
	return p, nil
}

func encryptKeys(keys, files []string, in io.Reader, out io.Writer, armor bool, yaml bool) {
	var recipients []age.Recipient
	for _, arg := range keys {
		r, err := parseRecipient(arg)
		if err != nil {
			logFatalf("Error: %v", err)
		}
		recipients = append(recipients, r)
	}
	for _, name := range files {
		recs, err := parseRecipientsFile(name)
		if err != nil {
			logFatalf("Error: failed to parse recipient file %q: %v", name, err)
		}
		recipients = append(recipients, recs...)
	}

	if yaml {
		encryptYAML(recipients, in, out)
	} else {
		encrypt(recipients, in, out, armor)
	}
}

func encryptPass(pass string, in io.Reader, out io.Writer, armor bool, yaml bool) {
	r, err := age.NewScryptRecipient(pass)
	if err != nil {
		logFatalf("Error: %v", err)
	}
	encrypt([]age.Recipient{r}, in, out, armor)

	if yaml {
		encryptYAML([]age.Recipient{r}, in, out)
	} else {
		encrypt([]age.Recipient{r}, in, out, armor)
	}
}

func encrypt(recipients []age.Recipient, in io.Reader, out io.Writer, withArmor bool) {
	if withArmor {
		a := armor.NewWriter(out)
		defer func() {
			if err := a.Close(); err != nil {
				logFatalf("Error: %v", err)
			}
		}()
		out = a
	}
	w, err := age.Encrypt(out, recipients...)
	if err != nil {
		logFatalf("Error: %v", err)
	}
	if _, err := io.Copy(w, in); err != nil {
		logFatalf("Error: %v", err)
	}
	if err := w.Close(); err != nil {
		logFatalf("Error: %v", err)
	}
}

func encryptYAML(recipients []age.Recipient, in io.Reader, out io.Writer) {
	node := yaml.Node{}
	w := yage.Wrapper{Value: &node}

	decoder := yaml.NewDecoder(in)
	encoder := yaml.NewEncoder(out)
	encoder.SetIndent(2)
	defer encoder.Close()

	for {
		err := decoder.Decode(&w)
		if err == io.EOF {
			break
		} else if err != nil {
			logFatalf("Error: %v", err)
		}

		// Encrypt the Nodes with the !crypto/age tag
		encNode, err := yage.MarshalYAML(&node, recipients)

		if err != nil {
			logFatalf("Error: %v", err)
		}

		err = encoder.Encode(&encNode)

		if err != nil {
			logFatalf("Error: %v", err)
		}
	}
}

func addOpenSSHIdentities(identities *[]age.Identity) {
	// If they exist and are well-formed, load the default SSH keys. If they are
	// passphrase protected, the passphrase will only be requested if the
	// identity matches a recipient stanza.
	for _, path := range []string{
		os.ExpandEnv("$HOME/.ssh/id_rsa"),
		os.ExpandEnv("$HOME/.ssh/id_ed25519"),
	} {
		content, err := ioutil.ReadFile(path)
		if err != nil {
			continue
		}
		ids, err := parseSSHIdentity(path, content)
		if err != nil {
			// If the key is explicitly requested, this error will be caught
			// below, otherwise ignore it silently.
			continue
		}
		*identities = append(*identities, ids...)
	}
}

func decrypt(keys []string, in io.Reader, out io.Writer) {
	identities := []age.Identity{
		// If there is an scrypt recipient (it will have to be the only one and)
		// this identity will be invoked.
		&LazyScryptIdentity{passphrasePrompt},
	}

	addOpenSSHIdentities(&identities)

	for _, name := range keys {
		ids, err := parseIdentitiesFile(name)
		if err != nil {
			logFatalf("Error reading %q: %v", name, err)
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
		logFatalf("Error: %v", err)
	}
	if _, err := io.Copy(out, r); err != nil {
		logFatalf("Error: %v", err)
	}
}

func decryptYAML(keys []string, in io.Reader, out io.Writer, discardNoTag bool) {
	identities := []age.Identity{
		// If there is an scrypt recipient (it will have to be the only one and)
		// this identity will be invoked.
		&LazyScryptIdentity{passphrasePrompt},
	}

	addOpenSSHIdentities(&identities)

	for _, name := range keys {
		ids, err := parseIdentitiesFile(name)
		if err != nil {
			logFatalf("Error reading %q: %v", name, err)
		}
		identities = append(identities, ids...)
	}

	node := yaml.Node{}
	w := yage.Wrapper{
		Value:        &node,
		Identities:   identities,
		DiscardNoTag: discardNoTag,
	}

	decoder := yaml.NewDecoder(in)
	encoder := yaml.NewEncoder(out)
	encoder.SetIndent(2)

	for {
		err := decoder.Decode(&w)
		if err == io.EOF {
			break
		} else if err != nil {
			logFatalf("Error: %v", err)
		}

		err = encoder.Encode(&node)

		if err != nil {
			logFatalf("Error: %v", err)
		}
	}

	encoder.Close()
}

func passphrasePrompt() (string, error) {
	fmt.Fprintf(os.Stderr, "Enter passphrase: ")
	pass, err := readPassphrase()
	if err != nil {
		return "", fmt.Errorf("could not read passphrase: %v", err)
	}
	return string(pass), nil
}

type lazyOpener struct {
	name      string
	overwrite bool
	f         *os.File
	err       error
}

func newLazyOpener(name string, overwrite bool) io.WriteCloser {
	return &lazyOpener{name: name, overwrite: overwrite}
}

func (l *lazyOpener) Write(p []byte) (n int, err error) {
	if l.f == nil && l.err == nil {
		oFlags := os.O_WRONLY | os.O_CREATE
		if l.overwrite {
			oFlags = oFlags | os.O_TRUNC
		} else {
			oFlags = oFlags | os.O_EXCL
		}
		l.f, l.err = os.OpenFile(l.name, oFlags, 0666)
	}
	if l.err != nil {
		return 0, l.err
	}
	return l.f.Write(p)
}

func (l *lazyOpener) Close() error {
	if l.f != nil {
		return l.f.Close()
	}
	return nil
}

func logFatalf(format string, v ...interface{}) {
	_log.Fatalf(format, v...)
}
