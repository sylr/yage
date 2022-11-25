// Copyright 2021 Google LLC
// Copyright 2021 Sylvain Rabot
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file or at
// https://developers.google.com/open-source/licenses/bsd

package yage

import (
	"github.com/spf13/cobra"

	"sylr.dev/yage/cmd/decrypt"
	"sylr.dev/yage/cmd/encrypt"
	"sylr.dev/yage/cmd/rekey"
)

var Version string = "dev"

var (
	decryptFlag bool
	encryptFlag bool
)

var YAGECmd = cobra.Command{
	Use:          "yage",
	Short:        "yage, yaml+age",
	Version:      Version,
	SilenceUsage: true,
}

func init() {
	YAGECmd.Flags().BoolVarP(&decryptFlag, "decrypt", "d", false, "decrypt data")
	YAGECmd.Flags().BoolVarP(&encryptFlag, "encrypt", "e", false, "encrypt data")

	if err := YAGECmd.Flags().MarkDeprecated("decrypt", "use decrypt sub-command instead"); err != nil {
		panic(err)
	}
	if err := YAGECmd.Flags().MarkDeprecated("encrypt", "use encrypt sub-command instead"); err != nil {
		panic(err)
	}

	YAGECmd.AddGroup(&cobra.Group{ID: "age", Title: "Commands:"})
	YAGECmd.AddCommand(&decrypt.DecryptCmd)
	YAGECmd.AddCommand(&encrypt.EncryptCmd)
	YAGECmd.AddCommand(&rekey.RekeyCmd)
}
