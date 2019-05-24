// Copyright © 2019 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/philips/sget/sget/github"
)

// githubCmd represents the github command
var githubCmd = &cobra.Command{
	Use:   "github",
	Short: "github subcommands",
	Long: `
public
.`,
}

func init() {
	rootCmd.AddCommand(githubCmd)
	github.AddCommands(githubCmd)

	githubCmd.PersistentFlags().String("foo", "", "A help for foo")

	githubCmd.PersistentFlags().StringP("owner", "o", "", "Repo owner name")
	viper.BindPFlag("owner", githubCmd.PersistentFlags().Lookup("owner"))

	githubCmd.PersistentFlags().StringP("repo", "r", "", "Repo name")
	viper.BindPFlag("repo", githubCmd.PersistentFlags().Lookup("repo"))

	githubCmd.PersistentFlags().StringP("tag", "t", "", "Release tag")
	viper.BindPFlag("tag", githubCmd.PersistentFlags().Lookup("tag"))

}
