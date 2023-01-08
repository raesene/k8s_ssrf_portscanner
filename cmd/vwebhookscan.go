/*
Copyright © 2022 Rory McCune <rorym@mccune.org.uk>

*/
package cmd

import (
	"math/rand"
	"net"
	"time"

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
		//fmt.Println("vwebhookscan called")
		options := cmd.Flags()
		//If the network flag is set we need to scan the network
		if options.Lookup("range").Value.String() != "" {
			ip, ipnet, err := net.ParseCIDR("192.168.41.0/24")
			if err != nil {
				panic(err)
			}
			for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
				options.Set("target", ip.String())
				// Lets try and make the namespace unique
				rand.Seed(time.Now().UnixNano())
				charset := "abcdefghijklmnopqrstuvwxyz"
				c := charset[rand.Intn(len(charset))]
				d := charset[rand.Intn(len(charset))]
				options.Set("namespace", "ssrfscanner"+string(c)+string(d))
				ssrfportscanner.VWebhookScan(options)
			}
		} else {
			// Lets try and make the namespace unique
			rand.Seed(time.Now().UnixNano())
			charset := "abcdefghijklmnopqrstuvwxyz"
			c := charset[rand.Intn(len(charset))]
			options.Set("namespace", "ssrfscanner"+string(c))
			ssrfportscanner.VWebhookScan(options)
		}
	},
}

func init() {
	rootCmd.AddCommand(vwebhookscanCmd)

}

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}
