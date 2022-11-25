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

var YAGECmd = cobra.Command{
	Use:     "yage",
	Short:   "yage, yaml+age",
	Version: Version,
}

func init() {
	YAGECmd.AddGroup(&cobra.Group{ID: "age", Title: "Commands:"})
	YAGECmd.AddCommand(&decrypt.DecryptCmd)
	YAGECmd.AddCommand(&encrypt.EncryptCmd)
	YAGECmd.AddCommand(&rekey.RekeyCmd)
}
