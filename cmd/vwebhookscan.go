/*
Copyright © 2022 Rory McCune <rorym@mccune.org.uk>

*/
package cmd

import (
	"fmt"

	"github.com/raesene/k8s_ssrf_portscanner/pkg/ssrfportscanner"
	"github.com/spf13/cobra"
)

// vwebhookscanCmd represents the vwebhookscan command
var vwebhookscanCmd = &cobra.Command{
	Use:   "vwebhookscan",
	Short: "use a validating webhook to scan a target",
	Long: `This command uses a validating webhook object to scan a target
	via modification of the URL parameter`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("vwebhookscan called")
		options := cmd.Flags()
		ssrfportscanner.VWebhookScan(options)
	},
}

func init() {
	rootCmd.AddCommand(vwebhookscanCmd)

}
